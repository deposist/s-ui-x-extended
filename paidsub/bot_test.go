package paidsub

import (
	"errors"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	cases := []struct {
		in      string
		wantCmd string
		wantArg string
	}{
		{"/start", "/start", ""},
		{"/start@MyBot code123", "/start", "code123"},
		{"/QR", "/qr", ""},
		{"  /stats  ", "/stats", ""},
		{"hello there", "", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		cmd, arg := parseCommand(tc.in)
		if cmd != tc.wantCmd || arg != tc.wantArg {
			t.Errorf("parseCommand(%q) = (%q,%q), want (%q,%q)", tc.in, cmd, arg, tc.wantCmd, tc.wantArg)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		1023:    "1023 B",
		1024:    "1.00 KiB",
		1536:    "1.50 KiB",
		1 << 20: "1.00 MiB",
		1 << 30: "1.00 GiB",
		-5:      "0 B",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	if got := progressBar(0); got != "[░░░░░░░░░░]" {
		t.Errorf("progressBar(0) = %q", got)
	}
	if got := progressBar(100); got != "[██████████]" {
		t.Errorf("progressBar(100) = %q", got)
	}
	if got := progressBar(150); got != "[██████████]" {
		t.Errorf("progressBar(150) clamp = %q", got)
	}
	if got := progressBar(50); got != "[█████░░░░░]" {
		t.Errorf("progressBar(50) = %q", got)
	}
}

func TestPickLang(t *testing.T) {
	if pickLang("ru") != langRU {
		t.Error("ru should map to langRU")
	}
	if pickLang("ru-RU") != langRU {
		t.Error("ru-RU should map to langRU")
	}
	if pickLang("en-US") != langEN {
		t.Error("en-US should map to langEN")
	}
	if pickLang("") != langEN {
		t.Error("empty should map to langEN")
	}
}

func TestChunkText(t *testing.T) {
	if got := chunkText("short", 100); len(got) != 1 || got[0] != "short" {
		t.Errorf("chunkText short = %v", got)
	}
	big := "aaaa\nbbbb\ncccc\ndddd"
	chunks := chunkText(big, 9)
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks, got %v", chunks)
	}
	for _, ch := range chunks {
		if len(ch) > 9 {
			t.Errorf("chunk too long: %q", ch)
		}
	}
	// A single line longer than max must be hard-split so no chunk exceeds max.
	for _, ch := range chunkText(strings.Repeat("x", 25), 10) {
		if len([]rune(ch)) > 10 {
			t.Errorf("oversized line not hard-split: %q (%d)", ch, len([]rune(ch)))
		}
	}
}

func TestFormatOrderAmount(t *testing.T) {
	if got := formatOrderAmount(5, "XTR"); got != "5 ⭐" {
		t.Errorf("XTR amount = %q, want %q", got, "5 ⭐")
	}
	if got := formatOrderAmount(10000, "RUB"); got != "100.00 RUB" {
		t.Errorf("RUB amount = %q, want %q", got, "100.00 RUB")
	}
}

func TestOrderStatusLabelFallback(t *testing.T) {
	if got := orderStatusLabel("paid", langEN); got != "paid" {
		t.Errorf("paid label = %q", got)
	}
	if got := orderStatusLabel("weird", langEN); got != "weird" {
		t.Errorf("unknown status should fall back to raw value, got %q", got)
	}
}

func callbackSet(kb *inlineKeyboard) map[string]bool {
	m := map[string]bool{}
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != "" {
				m[b.CallbackData] = true
			}
		}
	}
	return m
}

func TestMenuKeyboards(t *testing.T) {
	b := &Bot{}
	main := callbackSet(b.menuKeyboard(langEN))
	for _, want := range []string{"links", "qr", "stats", "payment", "awg:list", "help"} {
		if !main[want] {
			t.Errorf("main menu missing callback %q", want)
		}
	}
	// "buy" moved into the payment submenu; it must not be on the top level.
	if main["buy"] {
		t.Error("top-level menu should no longer expose 'buy'")
	}
	pay := callbackSet(b.paymentMenuKeyboard(langEN))
	for _, want := range []string{"buy", "orders", "refund", "menu"} {
		if !pay[want] {
			t.Errorf("payment submenu missing callback %q", want)
		}
	}
}

func TestAWGCallbackDataFitsTelegramLimit(t *testing.T) {
	for _, data := range []string{"awg:list", "awg:add", "awg:v:4294967295", "awg:c:4294967295", "awg:q:4294967295", "awg:r:4294967295", "awg:ry:4294967295:0123456789abcdef", "awg:d:4294967295", "awg:dy:4294967295"} {
		if awgCallbackTooLong(data) {
			t.Fatalf("callback exceeds Telegram limit: %q", data)
		}
	}
}

func TestAWGNameStateIsScopedExpiresAndClears(t *testing.T) {
	b := newBot()
	if !b.beginAWGName(10, 20, "request-a", 100) || !b.beginAWGName(10, 21, "request-b", 100) {
		t.Fatal("failed to begin AWG name state")
	}
	if _, ok := b.takeAWGNameState(11, 20, 101); ok {
		t.Fatal("state leaked to another Telegram user")
	}
	state, ok := b.takeAWGNameState(10, 20, 101)
	if !ok || state.requestKey != "request-a" {
		t.Fatalf("unexpected name state: %+v, %v", state, ok)
	}
	if _, ok := b.takeAWGNameState(10, 20, 101); ok {
		t.Fatal("consumed state was returned twice")
	}
	if _, ok := b.takeAWGNameState(10, 21, 100+awgNameStateTTL); ok {
		t.Fatal("expired state was accepted")
	}
	if !b.beginAWGName(10, 20, "request-c", 200) {
		t.Fatal("failed to replace state")
	}
	b.clearAWGNameStates()
	if _, ok := b.takeAWGNameState(10, 20, 201); ok {
		t.Fatal("state survived bot shutdown cleanup")
	}
}

func TestAWGRotationConfirmationUsesStablePerPromptNonce(t *testing.T) {
	first, ok := awgOperationNonce("callback-one")
	if !ok {
		t.Fatal("callback id was rejected")
	}
	replay, _ := awgOperationNonce("callback-one")
	second, _ := awgOperationNonce("callback-two")
	if first != replay {
		t.Fatal("callback replay did not produce the same nonce")
	}
	if first == second {
		t.Fatal("separate confirmations reused a nonce")
	}
	data := "awg:ry:42:" + first
	id, nonce, ok := parseAWGConfirmation(data, "awg:ry:")
	if !ok || id != 42 || nonce != first || awgCallbackTooLong(data) {
		t.Fatalf("confirmation callback did not round-trip: id=%d nonce=%q ok=%v", id, nonce, ok)
	}
	if _, _, ok := parseAWGConfirmation("awg:ry:42:not-hex", "awg:ry:"); ok {
		t.Fatal("malformed nonce was accepted")
	}
	firstKey, _ := awgOperationKey("tg-rotate", 7, 42, "callback-one")
	secondKey, _ := awgOperationKey("tg-rotate", 7, 42, "callback-two")
	if firstKey == secondKey {
		t.Fatal("separate rotation confirmations reused an idempotency key")
	}
}

func TestIsAlreadyRefunded(t *testing.T) {
	if !isAlreadyRefunded(&tgAPIError{Code: 400, Description: "Bad Request: CHARGE_ALREADY_REFUNDED"}) {
		t.Error("CHARGE_ALREADY_REFUNDED should be detected as already-refunded")
	}
	if isAlreadyRefunded(&tgAPIError{Code: 400, Description: "USER_NOT_FOUND"}) {
		t.Error("unrelated API error must not be treated as already-refunded")
	}
	if isAlreadyRefunded(nil) {
		t.Error("nil error is not already-refunded")
	}
	if isAlreadyRefunded(errors.New("network")) {
		t.Error("non-API error is not already-refunded")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(3, 60)
	now := int64(1000)
	for i := 0; i < 3; i++ {
		if !rl.allow(1, now) {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if rl.allow(1, now) {
		t.Fatal("4th request in window should be denied")
	}
	// A different key is independent.
	if !rl.allow(2, now) {
		t.Fatal("different key should be allowed")
	}
	// New window resets the count.
	if !rl.allow(1, now+60) {
		t.Fatal("request after window should be allowed")
	}
}
