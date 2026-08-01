package core

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

// TestNewFactoryIgnoresUnsafeLogOutput is the defense-in-depth half of the
// log.output confinement fix: even if an unsafe path reached the stored config
// (legacy import, manual DB edit), the core must not open it as a file.
func TestNewFactoryIgnoresUnsafeLogOutput(t *testing.T) {
	factory, err := NewFactory(log.Options{
		Options: option.LogOptions{Output: "/etc/cron.d/s-ui-pwn"},
	})
	if err != nil {
		t.Fatalf("NewFactory error: %v", err)
	}
	df, ok := factory.(*defaultFactory)
	if !ok {
		t.Fatalf("unexpected factory type %T", factory)
	}
	if df.filePath != "" {
		t.Fatalf("unsafe log.output should be ignored, got filePath %q", df.filePath)
	}
}

func TestNewFactoryKeepsSafeRelativeLogOutput(t *testing.T) {
	factory, err := NewFactory(log.Options{
		Options: option.LogOptions{Output: "box.log"},
	})
	if err != nil {
		t.Fatalf("NewFactory error: %v", err)
	}
	df, ok := factory.(*defaultFactory)
	if !ok {
		t.Fatalf("unexpected factory type %T", factory)
	}
	if df.filePath != "box.log" {
		t.Fatalf("safe relative path should be kept, got filePath %q", df.filePath)
	}
}

func TestObservableFactoryEmitsLogs(t *testing.T) {
	factory := NewDefaultFactory(context.Background(), log.Formatter{DisableTimestamp: true}, io.Discard, "")
	subscription, done, err := factory.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer factory.UnSubscribe(subscription)

	factory.NewLogger("test").Info("hello")
	select {
	case entry := <-subscription:
		if entry.Level != log.LevelInfo || entry.Message != "test: hello" {
			t.Fatalf("unexpected log entry: %+v", entry)
		}
	case <-done:
		t.Fatal("log subscription closed before receiving an entry")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for observable log entry")
	}
}

func TestObservableLoggerPanicPreservesPanicSemantics(t *testing.T) {
	factory := NewDefaultFactory(context.Background(), log.Formatter{DisableTimestamp: true}, io.Discard, "")
	defer func() {
		value := recover()
		if value == nil || !strings.Contains(value.(string), "panic-message") {
			t.Fatalf("panic value = %v", value)
		}
	}()
	factory.Logger().Panic("panic-message")
}
