package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func TestLoadTokensMigratesLegacyPlaintextToken(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}

	if err := database.GetDB().Create(&model.Tokens{
		Desc:    "legacy",
		Token:   "legacy-token",
		Enabled: true,
		Expiry:  0,
		UserId:  1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	raw, err := userService.LoadTokens()
	if err != nil {
		t.Fatal(err)
	}
	var loaded []map[string]any
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected one loaded token, got %d", len(loaded))
	}
	if loaded[0]["tokenHash"] == "" || loaded[0]["token"] != nil {
		t.Fatalf("loaded token leaked plaintext or missed hash: %#v", loaded[0])
	}

	var stored model.Tokens
	if err := database.GetDB().First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Token != "" {
		t.Fatalf("legacy plaintext token was not cleared: %q", stored.Token)
	}
	if stored.TokenHash == "" || stored.TokenPrefix != tokenPrefix("legacy-token") {
		t.Fatalf("legacy token hash/prefix not populated: %#v", stored)
	}
	if !stored.Enabled || stored.Scope != defaultAPITokenScope {
		t.Fatalf("legacy token defaults not populated: %#v", stored)
	}
}

func TestAddTokenValidatesScopeAllowlist(t *testing.T) {
	initSettingTestDB(t)
	userService := &UserService{}

	for _, scope := range []string{"", "admin", "read", "write", "database", "telegram", "observability"} {
		if _, err := userService.AddToken("admin", 0, "valid "+scope, scope); err != nil {
			t.Fatalf("scope %q should be accepted: %v", scope, err)
		}
	}
	if _, err := userService.AddToken("admin", 0, "invalid", "full"); err == nil {
		t.Fatal("scope full should be rejected")
	}
	if _, err := userService.AddToken("admin", 0, "invalid", "admin "); err != nil {
		t.Fatalf("trimmed admin scope should be accepted: %v", err)
	}

	var tokens []model.Tokens
	if err := database.GetDB().Order("id asc").Find(&tokens).Error; err != nil {
		t.Fatal(err)
	}
	for _, token := range tokens {
		if !apiTokenScopeAllowed(token.Scope) {
			t.Fatalf("stored invalid scope: %#v", token)
		}
	}
}

func TestTokenUseDebouncerKeepsLatestUseUntilFlush(t *testing.T) {
	written := make(chan map[uint]tokenUseUpdate, 1)
	debouncer := newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		copied := make(map[uint]tokenUseUpdate, len(updates))
		for id, update := range updates {
			copied[id] = update
		}
		written <- copied
		return nil
	})

	debouncer.Record(7, "198.51.100.1", 100)
	debouncer.Record(7, "198.51.100.2", 200)
	if err := debouncer.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}

	updates := <-written
	if len(updates) != 1 {
		t.Fatalf("expected one debounced token update, got %#v", updates)
	}
	if updates[7].ip != "198.51.100.2" || updates[7].ts != 200 {
		t.Fatalf("debouncer did not keep latest token use: %#v", updates[7])
	}
}

func TestTokenUseDebouncerTimerFailureOpensCircuitIssue28(t *testing.T) {
	resumeTokenUseFlush()
	t.Cleanup(resumeTokenUseFlush)
	errFlush := errors.New("timer flush failed")
	attempts := 0
	var successful map[uint]tokenUseUpdate
	debouncer := newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		attempts++
		copied := make(map[uint]tokenUseUpdate, len(updates))
		for id, update := range updates {
			copied[id] = update
		}
		if attempts == 1 {
			return errFlush
		}
		successful = copied
		return nil
	})
	debouncer.failureBackoff = time.Hour
	t.Cleanup(func() {
		stopTokenUseDebouncerTimer(debouncer)
	})

	debouncer.Record(7, "198.51.100.1", 100)
	epoch := stopTokenUseDebouncerTimer(debouncer)
	debouncer.flushTimer(epoch)
	if attempts != 1 {
		t.Fatalf("expected first timer write attempt, got %d", attempts)
	}

	debouncer.mu.Lock()
	if len(debouncer.pending) != 1 {
		t.Fatalf("failed update was not requeued: %#v", debouncer.pending)
	}
	if !debouncer.circuitUntil.After(time.Now()) {
		t.Fatalf("circuit was not opened after timer failure: %v", debouncer.circuitUntil)
	}
	debouncer.mu.Unlock()

	debouncer.Record(8, "198.51.100.8", 300)
	epoch = stopTokenUseDebouncerTimer(debouncer)
	debouncer.flushTimer(epoch)
	if attempts != 1 {
		t.Fatalf("open circuit should block immediate retry, got %d attempts", attempts)
	}

	debouncer.mu.Lock()
	debouncer.circuitUntil = time.Now().Add(-time.Second)
	epoch = debouncer.epoch
	if debouncer.timer != nil {
		debouncer.timer.Stop()
		debouncer.timer = nil
	}
	debouncer.mu.Unlock()
	debouncer.flushTimer(epoch)

	if attempts != 2 {
		t.Fatalf("expected retry after circuit cooldown, got %d attempts", attempts)
	}
	if len(successful) != 2 {
		t.Fatalf("expected requeued and newer pending updates, got %#v", successful)
	}
	if successful[7].ip != "198.51.100.1" || successful[7].ts != 100 {
		t.Fatalf("failed update was not retried: %#v", successful[7])
	}
	if successful[8].ip != "198.51.100.8" || successful[8].ts != 300 {
		t.Fatalf("new pending update was not included in retry: %#v", successful[8])
	}

	debouncer.mu.Lock()
	circuitUntil := debouncer.circuitUntil
	pending := len(debouncer.pending)
	debouncer.mu.Unlock()
	if !circuitUntil.IsZero() {
		t.Fatalf("successful timer flush did not close circuit: %v", circuitUntil)
	}
	if pending != 0 {
		t.Fatalf("pending updates remain after successful retry: %d", pending)
	}
}

func TestTokenUseDebouncerKeepsLatestAfterTimerFailureIssue28(t *testing.T) {
	resumeTokenUseFlush()
	t.Cleanup(resumeTokenUseFlush)
	errFlush := errors.New("timer flush failed")
	attempts := 0
	var successful map[uint]tokenUseUpdate
	var debouncer *tokenUseDebouncer
	debouncer = newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		attempts++
		copied := make(map[uint]tokenUseUpdate, len(updates))
		for id, update := range updates {
			copied[id] = update
		}
		if attempts == 1 {
			debouncer.Record(7, "198.51.100.2", 200)
			return errFlush
		}
		successful = copied
		return nil
	})
	debouncer.failureBackoff = time.Hour
	t.Cleanup(func() {
		stopTokenUseDebouncerTimer(debouncer)
	})

	debouncer.Record(7, "198.51.100.1", 100)
	epoch := stopTokenUseDebouncerTimer(debouncer)
	debouncer.flushTimer(epoch)

	debouncer.mu.Lock()
	if pending := debouncer.pending[7]; pending.ip != "198.51.100.2" || pending.ts != 200 {
		t.Fatalf("newer pending update did not win after requeue: %#v", pending)
	}
	debouncer.circuitUntil = time.Now().Add(-time.Second)
	epoch = debouncer.epoch
	if debouncer.timer != nil {
		debouncer.timer.Stop()
		debouncer.timer = nil
	}
	debouncer.mu.Unlock()
	debouncer.flushTimer(epoch)

	if attempts != 2 {
		t.Fatalf("expected one failed attempt and one retry, got %d", attempts)
	}
	if len(successful) != 1 {
		t.Fatalf("expected one latest token update, got %#v", successful)
	}
	if successful[7].ip != "198.51.100.2" || successful[7].ts != 200 {
		t.Fatalf("retry wrote stale token use: %#v", successful[7])
	}
}

func TestTokenUseDebouncerTimerFailureInvalidatesConcurrentNormalTimerIssue28(t *testing.T) {
	resumeTokenUseFlush()
	t.Cleanup(resumeTokenUseFlush)
	errFlush := errors.New("timer flush failed")
	attempts := 0
	var successful map[uint]tokenUseUpdate
	var debouncer *tokenUseDebouncer
	debouncer = newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		attempts++
		copied := make(map[uint]tokenUseUpdate, len(updates))
		for id, update := range updates {
			copied[id] = update
		}
		if attempts == 1 {
			debouncer.Record(8, "198.51.100.8", 300)
			return errFlush
		}
		successful = copied
		return nil
	})
	debouncer.failureBackoff = 2 * time.Hour
	t.Cleanup(func() {
		stopTokenUseDebouncerTimer(debouncer)
	})

	debouncer.Record(7, "198.51.100.1", 100)
	staleEpoch := stopTokenUseDebouncerTimer(debouncer)
	debouncer.flushTimer(staleEpoch)
	if attempts != 1 {
		t.Fatalf("expected first timer write attempt, got %d", attempts)
	}

	debouncer.mu.Lock()
	if debouncer.epoch == staleEpoch {
		t.Fatalf("failed write did not invalidate stale timer epoch %d", staleEpoch)
	}
	if len(debouncer.pending) != 2 {
		t.Fatalf("expected failed and concurrent updates to remain pending: %#v", debouncer.pending)
	}
	if debouncer.timer == nil {
		t.Fatal("expected retry timer after failed write")
	}
	circuitUntil := debouncer.circuitUntil
	debouncer.mu.Unlock()
	if !circuitUntil.After(time.Now()) {
		t.Fatalf("expected open circuit after failed write: %v", circuitUntil)
	}

	debouncer.flushTimer(staleEpoch)
	if attempts != 1 {
		t.Fatalf("stale normal timer callback retried write, attempts=%d", attempts)
	}

	debouncer.Record(9, "198.51.100.9", 400)
	debouncer.mu.Lock()
	if !debouncer.circuitUntil.Equal(circuitUntil) {
		t.Fatalf("Record shortened or changed open circuit: before %v after %v", circuitUntil, debouncer.circuitUntil)
	}
	retryEpoch := debouncer.epoch
	debouncer.mu.Unlock()

	debouncer.flushTimer(retryEpoch)
	if attempts != 1 {
		t.Fatalf("open circuit allowed immediate retry, attempts=%d", attempts)
	}

	debouncer.mu.Lock()
	debouncer.circuitUntil = time.Now().Add(-time.Second)
	retryEpoch = debouncer.epoch
	if debouncer.timer != nil {
		debouncer.timer.Stop()
		debouncer.timer = nil
	}
	debouncer.mu.Unlock()
	debouncer.flushTimer(retryEpoch)
	if attempts != 2 {
		t.Fatalf("expected retry after cooldown, got %d attempts", attempts)
	}
	if len(successful) != 3 {
		t.Fatalf("expected all bounded pending updates on retry, got %#v", successful)
	}
	for id, expected := range map[uint]tokenUseUpdate{
		7: {ip: "198.51.100.1", ts: 100},
		8: {ip: "198.51.100.8", ts: 300},
		9: {ip: "198.51.100.9", ts: 400},
	} {
		if successful[id] != expected {
			t.Fatalf("retry update %d = %#v, want %#v", id, successful[id], expected)
		}
	}
}

func TestTokenUseDebouncerManualFlushBypassesCircuitIssue28(t *testing.T) {
	resumeTokenUseFlush()
	t.Cleanup(resumeTokenUseFlush)
	attempts := 0
	debouncer := newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		attempts++
		if len(updates) != 1 || updates[7].ip != "198.51.100.7" || updates[7].ts != 700 {
			t.Fatalf("manual flush wrote unexpected updates: %#v", updates)
		}
		return nil
	})
	t.Cleanup(func() {
		stopTokenUseDebouncerTimer(debouncer)
	})

	debouncer.mu.Lock()
	debouncer.circuitUntil = time.Now().Add(time.Hour)
	debouncer.mu.Unlock()
	debouncer.Record(7, "198.51.100.7", 700)
	if err := debouncer.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("manual flush did not bypass open circuit, attempts=%d", attempts)
	}
	debouncer.mu.Lock()
	circuitUntil := debouncer.circuitUntil
	pending := len(debouncer.pending)
	timer := debouncer.timer
	debouncer.mu.Unlock()
	if !circuitUntil.IsZero() {
		t.Fatalf("manual flush success did not close circuit: %v", circuitUntil)
	}
	if pending != 0 {
		t.Fatalf("manual flush left pending updates: %d", pending)
	}
	if timer != nil {
		t.Fatal("manual flush left retry timer after draining pending updates")
	}
}

func TestTokenUseDebouncerForceFlushFailureDoesNotRetryIssue28(t *testing.T) {
	resumeTokenUseFlush()
	t.Cleanup(resumeTokenUseFlush)
	errFlush := errors.New("force flush failed")
	debouncer := newTokenUseDebouncer(time.Hour, func(updates map[uint]tokenUseUpdate) error {
		return errFlush
	})
	t.Cleanup(func() {
		stopTokenUseDebouncerTimer(debouncer)
	})

	debouncer.Record(7, "198.51.100.7", 700)
	if err := debouncer.flushNow(context.Background(), true); !errors.Is(err, errFlush) {
		t.Fatalf("force flush error = %v, want %v", err, errFlush)
	}
	debouncer.mu.Lock()
	pending := len(debouncer.pending)
	circuitUntil := debouncer.circuitUntil
	timer := debouncer.timer
	debouncer.mu.Unlock()
	if pending != 0 {
		t.Fatalf("force flush failure requeued pending updates: %d", pending)
	}
	if !circuitUntil.IsZero() {
		t.Fatalf("force flush failure opened circuit: %v", circuitUntil)
	}
	if timer != nil {
		t.Fatal("force flush failure scheduled retry timer")
	}
}

func stopTokenUseDebouncerTimer(debouncer *tokenUseDebouncer) uint64 {
	debouncer.mu.Lock()
	defer debouncer.mu.Unlock()
	if debouncer.timer != nil {
		debouncer.timer.Stop()
		debouncer.timer = nil
	}
	return debouncer.epoch
}

func TestRecordTokenUseFlushesBatchedUpdate(t *testing.T) {
	initSettingTestDB(t)
	resetTokenUseDebouncerForTest()
	t.Cleanup(resetTokenUseDebouncerForTest)

	userService := &UserService{}
	if err := database.GetDB().Create(&model.Tokens{
		Desc:      "tracked",
		TokenHash: "hash",
		Enabled:   true,
		UserId:    1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	var token model.Tokens
	if err := database.GetDB().Where("desc = ?", "tracked").First(&token).Error; err != nil {
		t.Fatal(err)
	}

	if err := userService.RecordTokenUse(token.Id, "198.51.100.10"); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().First(&token, token.Id).Error; err != nil {
		t.Fatal(err)
	}
	if token.LastUsedAt != 0 || token.LastUsedIP != "" {
		t.Fatalf("token use was written before flush: %#v", token)
	}

	if err := userService.RecordTokenUse(token.Id, "198.51.100.11"); err != nil {
		t.Fatal(err)
	}
	if err := StopTokenUseDebouncer(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().First(&token, token.Id).Error; err != nil {
		t.Fatal(err)
	}
	if token.LastUsedAt == 0 || token.LastUsedIP != "198.51.100.11" {
		t.Fatalf("token use was not flushed with latest value: %#v", token)
	}
}
