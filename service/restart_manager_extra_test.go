package service

import (
	"testing"
	"time"
)

func TestRestartManagerExtraScheduleDedupesPendingRun(t *testing.T) {
	called := make(chan struct{}, 2)
	manager := newRestartManager(time.Hour, func() error {
		called <- struct{}{}
		return nil
	})

	if err := manager.ScheduleRestart(time.Hour); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	firstTimer := manager.pendingTimer
	manager.mu.Unlock()
	if firstTimer == nil {
		t.Fatal("first schedule did not create pending timer")
	}
	if err := manager.ScheduleRestart(time.Millisecond); err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	secondTimer := manager.pendingTimer
	inFlight := manager.inFlight
	manager.mu.Unlock()
	if secondTimer != firstTimer || !inFlight {
		t.Fatalf("second schedule should keep existing pending timer: same=%v inFlight=%v", secondTimer == firstTimer, inFlight)
	}
	manager.cancelPending()

	select {
	case <-called:
		t.Fatal("deduped pending restart should not fire during test")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRestartManagerExtraCancelPendingIsIdempotent(t *testing.T) {
	called := make(chan struct{}, 1)
	manager := newRestartManager(time.Hour, func() error {
		called <- struct{}{}
		return nil
	})

	if err := manager.ScheduleRestart(100 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	manager.cancelPending()
	manager.cancelPending()
	if err := manager.run(func() error { return nil }); err != nil {
		t.Fatal(err)
	}

	select {
	case <-called:
		t.Fatal("canceled restart fired")
	case <-time.After(150 * time.Millisecond):
	}
}
