package cronjob

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestCronJobStartRegistersJobsSynchronously(t *testing.T) {
	initCronJobTestDB(t)

	c := NewCronJob()
	if err := c.Start(time.UTC, 30); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop() })

	entries := c.cron.Entries()
	if len(entries) != 13 {
		t.Fatalf("expected 13 registered cron entries immediately after Start, got %d", len(entries))
	}
}

func TestCronJobStopWaitsForRunningJob(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	c := NewCronJob()
	c.cron = cron.New(cron.WithSeconds())
	if _, err := c.cron.AddFunc("@every 1s", func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	c.cron.Start()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("cron job did not start")
	}

	done := make(chan error, 1)
	go func() { done <- c.Stop() }()
	select {
	case err := <-done:
		t.Fatalf("Stop returned before running job completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not await running job")
	}
}

func TestCronJobStartWaitsForPreviousGeneration(t *testing.T) {
	initCronJobTestDB(t)
	started := make(chan struct{})
	release := make(chan struct{})
	c := NewCronJob()
	c.cron = cron.New(cron.WithSeconds())
	if _, err := c.cron.AddFunc("@every 1s", func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	c.cron.Start()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("old cron generation did not start")
	}

	done := make(chan error, 1)
	go func() { done <- c.Start(time.UTC, 0) }()
	select {
	case err := <-done:
		t.Fatalf("new generation started before old generation stopped: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("new generation did not start after old generation stopped")
	}
	t.Cleanup(func() { _ = c.Stop() })
}

func TestCronJobStopTimesOut(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	c := NewCronJob()
	c.stopTimeout = 25 * time.Millisecond
	c.cron = cron.New(cron.WithSeconds())
	if _, err := c.cron.AddFunc("@every 1s", func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	c.cron.Start()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("cron job did not start")
	}
	if err := c.Stop(); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error = %v, want deadline exceeded", err)
	}
	close(release)
}
