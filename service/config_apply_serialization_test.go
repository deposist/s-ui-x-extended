package service

import (
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// startTestCore boots a minimal real sing-box core or skips the test when the
// environment cannot start one (mirrors the existing hot-reload regressions).
func startTestCore(t *testing.T) *core.Core {
	t.Helper()
	coreInstance := core.NewCore()
	if err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)); err != nil {
		t.Skipf("minimal core start unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})
	return coreInstance
}

// A post-save apply must wait for an in-flight core operation instead of being
// silently skipped - otherwise the core keeps serving stale configuration.
func TestConfigSaveApplyNotSkippedDuringConcurrentRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     "mixed-apply-serialized",
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inbounds, err := json.Marshal([]uint{inbound.Id})
	if err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:      true,
		Name:        "alice",
		Config:      json.RawMessage(`{"mixed":{"username":"alice","password":"old"}}`),
		Inbounds:    inbounds,
		Links:       json.RawMessage(`[]`),
		IPLimitMode: "monitor",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var reloaded []uint
	prevRestart := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, inboundIds []uint) error {
		mu.Lock()
		reloaded = append([]uint(nil), inboundIds...)
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() {
		restartInboundsAfterSave = prevRestart
	})

	runtime := NewRuntime(coreInstance)
	manager := runtime.restart()
	opStarted := make(chan struct{})
	opRelease := make(chan struct{})
	opDone := make(chan error, 1)
	go func() {
		opDone <- manager.run(func() error {
			close(opStarted)
			<-opRelease
			return nil
		})
	}()
	<-opStarted

	client.Config = json.RawMessage(`{"mixed":{"username":"alice","password":"new"}}`)
	payload, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}
	saveDone := make(chan error, 1)
	go func() {
		configService := NewConfigServiceWithRuntime(runtime)
		_, saveErr := configService.Save("clients", "edit", payload, "", "admin", "example.com")
		saveDone <- saveErr
	}()

	select {
	case <-saveDone:
		t.Fatal("Save returned while a core operation was in flight - post-commit apply did not wait")
	case <-time.After(100 * time.Millisecond):
	}

	close(opRelease)
	if err := <-opDone; err != nil {
		t.Fatal(err)
	}
	if err := <-saveDone; err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(reloaded, []uint{inbound.Id}) {
		t.Fatalf("post-save reload skipped during concurrent restart; reloaded=%v want [%d]", reloaded, inbound.Id)
	}
}

func seedConfigBlob(t *testing.T, blob json.RawMessage) {
	t.Helper()
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	if err := (&SettingService{}).SaveConfig(tx, blob); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
}

func TestConfigSaveIdenticalConfigSkipsCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	blob := json.RawMessage(`{"log":{"disabled":true}}`)
	seedConfigBlob(t, blob)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("config", "set", blob, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if coreInstance.GetInstance() != before {
		t.Fatal("byte-identical config re-save must not restart the core")
	}
}

func TestConfigSaveChangedConfigRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	changed := json.RawMessage(`{"log":{"disabled":true,"level":"warn"}}`)
	if _, err := configService.Save("config", "set", changed, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("changed config blob must restart the core")
	}
	if after == nil {
		t.Fatal("core did not come back up after config change restart")
	}
}
