package realtime

import (
	"sync"
	"testing"
	"time"
)

func TestHubPublishFansOutAndGatesSecurityEvents(t *testing.T) {
	h := newHub()
	adminCh := make(chan Event, 2)
	readCh := make(chan Event, 2)
	unregisterAdmin := h.Register(&ClientHandle{User: "admin", Scope: ScopeAdmin, SendCh: adminCh})
	defer unregisterAdmin()
	unregisterRead := h.Register(&ClientHandle{User: "reader", Scope: ScopeRead, SendCh: readCh})
	defer unregisterRead()

	h.Publish(TopicOnlines, map[string]any{"count": 1})
	expectEvent(t, adminCh, TopicOnlines)
	expectEvent(t, readCh, TopicOnlines)

	h.Publish(TopicSecurityEvent, map[string]any{"kind": "test"})
	expectEvent(t, adminCh, TopicSecurityEvent)
	expectNoEvent(t, readCh)
}

func TestHubSlowUnbufferedClientIsDropped(t *testing.T) {
	h := newHub()
	sendCh := make(chan Event)
	drops := make(chan string, 1)
	h.Register(&ClientHandle{
		User:   "admin",
		Scope:  ScopeAdmin,
		SendCh: sendCh,
		OnDrop: func(reason string) {
			drops <- reason
		},
	})

	h.Publish(TopicOnlines, nil)
	select {
	case reason := <-drops:
		if reason != "slow" {
			t.Fatalf("unexpected drop reason: %s", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("slow client was not dropped")
	}

	h.Publish(TopicOnlines, nil)
	expectNoString(t, drops)
}

func TestHubCloseAllDropsClientsAndStopsDelivery(t *testing.T) {
	h := newHub()
	sendCh := make(chan Event, 1)
	drops := make(chan string, 1)
	h.Register(&ClientHandle{
		User:   "admin",
		Scope:  ScopeAdmin,
		SendCh: sendCh,
		OnDrop: func(reason string) {
			drops <- reason
		},
	})

	h.CloseAll("session_rotated")
	select {
	case reason := <-drops:
		if reason != "session_rotated" {
			t.Fatalf("unexpected close reason: %s", reason)
		}
	case <-time.After(time.Second):
		t.Fatal("client was not dropped on CloseAll")
	}

	h.Publish(TopicOnlines, nil)
	expectNoEvent(t, sendCh)
}

func TestHubUnregisterIsIdempotent(t *testing.T) {
	h := newHub()
	sendCh := make(chan Event, 1)
	unregister := h.Register(&ClientHandle{User: "admin", Scope: ScopeAdmin, SendCh: sendCh})
	unregister()
	unregister()

	h.Publish(TopicOnlines, nil)
	expectNoEvent(t, sendCh)
}

func TestHubReserveEnforcesUserAndIPLimits(t *testing.T) {
	h := newHub()
	releases := make([]func(), 0, 2)
	for i := 0; i < 2; i++ {
		release, ok := h.Reserve("admin", "192.0.2.1", 2, 10)
		if !ok {
			t.Fatalf("reservation %d should have succeeded", i)
		}
		releases = append(releases, release)
	}
	if _, ok := h.Reserve("admin", "192.0.2.2", 2, 10); ok {
		t.Fatal("reservation over user limit should fail")
	}
	releases[0]()
	releases[0]()
	if release, ok := h.Reserve("admin", "192.0.2.2", 2, 10); ok {
		release()
	} else {
		t.Fatal("reservation after release should succeed")
	}

	if release, ok := h.Reserve("reader", "198.51.100.1", 10, 1); ok {
		defer release()
	} else {
		t.Fatal("first IP reservation should succeed")
	}
	if _, ok := h.Reserve("writer", "198.51.100.1", 10, 1); ok {
		t.Fatal("reservation over IP limit should fail")
	}
}

func TestHubConcurrentRegisterPublishCloseWithUnbufferedChannels(t *testing.T) {
	h := newHub()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendCh := make(chan Event)
			unregister := h.Register(&ClientHandle{User: "admin", Scope: ScopeAdmin, SendCh: sendCh})
			defer unregister()
			for j := 0; j < 64; j++ {
				h.Publish(TopicOnlines, nil)
				if j%8 == 0 {
					h.CloseAll("test")
				}
			}
		}()
	}
	wg.Wait()
}

func expectEvent(t *testing.T, ch <-chan Event, topic Topic) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != topic {
			t.Fatalf("expected topic %s, got %s", topic, event.Type)
		}
		if event.Ts == 0 {
			t.Fatal("event timestamp was not set")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", topic)
	}
}

func expectNoEvent(t *testing.T, ch <-chan Event) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("unexpected event: %#v", event)
	case <-time.After(25 * time.Millisecond):
	}
}

func expectNoString(t *testing.T, ch <-chan string) {
	t.Helper()
	select {
	case value := <-ch:
		t.Fatalf("unexpected value: %s", value)
	case <-time.After(25 * time.Millisecond):
	}
}
