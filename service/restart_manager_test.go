package service

import (
	"testing"
	"time"
)

func TestRestartManagerDedupesInFlightOperation(t *testing.T) {
	manager := newRestartManager(time.Hour, func() error { return nil })
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- manager.run(func() error {
			close(started)
			<-release
			return nil
		})
	}()

	<-started
	ran := false
	if err := manager.run(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if ran {
		t.Fatal("second operation ran while first operation was in flight")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := manager.run(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("operation did not run after in-flight operation completed")
	}
}

func TestRestartManagerCancelPendingSighup(t *testing.T) {
	called := make(chan struct{}, 1)
	manager := newRestartManager(50*time.Millisecond, func() error {
		called <- struct{}{}
		return nil
	})

	if err := manager.sendSighup(); err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := manager.run(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if ran {
		t.Fatal("operation ran while delayed SIGHUP was pending")
	}

	manager.cancelPending()
	if err := manager.run(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("operation did not run after pending SIGHUP was canceled")
	}

	select {
	case <-called:
		t.Fatal("delayed SIGHUP signal ran after cancellation")
	case <-time.After(75 * time.Millisecond):
	}
}

type fakeRestartScheduler struct {
	delay time.Duration
}

func (s *fakeRestartScheduler) ScheduleRestart(delay time.Duration) error {
	s.delay = delay
	return nil
}

func TestPanelServiceUsesInjectedRestartScheduler(t *testing.T) {
	scheduler := &fakeRestartScheduler{}
	panel := NewPanelService(scheduler)

	if err := panel.RestartPanel(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	if scheduler.delay != 3*time.Second {
		t.Fatalf("scheduled delay = %s, want 3s", scheduler.delay)
	}
}
