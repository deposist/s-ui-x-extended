package service

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x/realtime"
)

func TestRotateSessionGenerationClosesRealtimeSessions(t *testing.T) {
	settingService := initSettingTestDB(t)
	realtime.CloseAll("test_reset")
	drops := make(chan string, 1)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: make(chan realtime.Event, 1),
		OnDrop: func(reason string) {
			drops <- reason
		},
	})
	defer unregister()

	if _, err := settingService.RotateSessionGeneration(); err != nil {
		t.Fatal(err)
	}

	select {
	case reason := <-drops:
		if reason != "session_rotated" {
			t.Fatalf("unexpected close reason: %s", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("realtime sessions were not closed")
	}
}
