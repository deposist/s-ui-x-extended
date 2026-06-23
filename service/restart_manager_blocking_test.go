package service

import (
	"testing"
	"time"
)

func TestRestartManagerRunBlockingWaitsForInFlightOperation(t *testing.T) {
	manager := newRestartManager(time.Hour, func() error { return nil })
	started := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- manager.run(func() error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	ran := make(chan struct{}, 1)
	blockingDone := make(chan error, 1)
	go func() {
		blockingDone <- manager.runBlocking(func() error {
			ran <- struct{}{}
			return nil
		})
	}()

	select {
	case <-ran:
		t.Fatal("runBlocking executed while another operation was in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-blockingDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-ran:
	default:
		t.Fatal("runBlocking operation never ran after the in-flight operation finished")
	}
}

func TestRestartManagerRunBlockingWaitsForPendingSighup(t *testing.T) {
	signaled := make(chan struct{})
	manager := newRestartManager(30*time.Millisecond, func() error {
		close(signaled)
		return nil
	})

	if err := manager.sendSighup(); err != nil {
		t.Fatal(err)
	}

	ran := false
	if err := manager.runBlocking(func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("runBlocking operation did not run after the pending SIGHUP fired")
	}
	select {
	case <-signaled:
	default:
		t.Fatal("runBlocking proceeded before the pending SIGHUP fired")
	}
}
