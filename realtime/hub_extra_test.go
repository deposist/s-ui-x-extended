package realtime

import (
	"testing"
	"time"
)

func TestHubExtraReserveRegisterDropAndAuthGating(t *testing.T) {
	h := newHub()
	release, ok := h.Reserve("admin", "203.0.113.10", 1, 1)
	if !ok {
		t.Fatal("first reservation should succeed")
	}
	if _, ok := h.Reserve("admin", "203.0.113.10", 1, 1); ok {
		t.Fatal("second reservation should be rejected by user/IP capacity")
	}
	release()
	if releaseAgain, ok := h.Reserve("admin", "203.0.113.10", 1, 1); ok {
		releaseAgain()
	} else {
		t.Fatal("reservation should be available after release")
	}

	adminCh := make(chan Event, 2)
	readCh := make(chan Event, 2)
	slowCh := make(chan Event)
	drops := make(chan string, 1)
	h.Register(&ClientHandle{User: "admin", Scope: ScopeAdmin, SendCh: adminCh})
	h.Register(&ClientHandle{User: "reader", Scope: ScopeRead, SendCh: readCh})
	h.Register(&ClientHandle{
		User:   "slow",
		Scope:  ScopeAdmin,
		SendCh: slowCh,
		OnDrop: func(reason string) {
			drops <- reason
		},
	})

	h.Publish(TopicSecurityEvent, "security")
	expectExtraHubEvent(t, adminCh, TopicSecurityEvent)
	expectExtraHubNoEvent(t, readCh)
	select {
	case reason := <-drops:
		if reason != "slow" {
			t.Fatalf("unexpected slow drop reason %q", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("slow client was not dropped")
	}

	h.Publish(TopicOnlines, "public")
	expectExtraHubEvent(t, adminCh, TopicOnlines)
	expectExtraHubEvent(t, readCh, TopicOnlines)
}

func expectExtraHubEvent(t *testing.T, ch <-chan Event, topic Topic) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != topic {
			t.Fatalf("expected topic %s, got %s", topic, event.Type)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", topic)
	}
}

func expectExtraHubNoEvent(t *testing.T, ch <-chan Event) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("unexpected event: %#v", event)
	case <-time.After(25 * time.Millisecond):
	}
}
