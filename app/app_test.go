package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestStartLifecycleRollsBackStartedComponentsInReverseOrder(t *testing.T) {
	var calls []string
	errExpected := errors.New("sub start failed")
	err := runStartLifecycle([]lifecycleStep{
		{name: "awg", start: func() error { calls = append(calls, "start awg"); return nil }, stop: func() { calls = append(calls, "stop awg") }},
		{name: "cron", start: func() error { calls = append(calls, "start cron"); return nil }, stop: func() { calls = append(calls, "stop cron") }},
		{name: "web", start: func() error { calls = append(calls, "start web"); return nil }, stop: func() { calls = append(calls, "stop web") }},
		{name: "sub", start: func() error { calls = append(calls, "start sub"); return errExpected }, stop: func() { calls = append(calls, "stop sub") }},
	})
	if !errors.Is(err, errExpected) {
		t.Fatalf("Start error = %v, want %v", err, errExpected)
	}
	want := []string{"start awg", "start cron", "start web", "start sub", "stop web", "stop cron", "stop awg"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("lifecycle calls = %v, want %v", calls, want)
	}
}

func TestStartLifecycleDoesNotStopComponentsThatDidNotStart(t *testing.T) {
	var stopped bool
	errExpected := errors.New("awg start failed")
	err := runStartLifecycle([]lifecycleStep{{
		name:  "awg",
		start: func() error { return errExpected },
		stop:  func() { stopped = true },
	}})
	if !errors.Is(err, errExpected) {
		t.Fatalf("Start error = %v, want %v", err, errExpected)
	}
	if stopped {
		t.Fatal("failed component was rolled back although it never started")
	}
}

func TestAWGLoopRepeatedStartStopOwnsOneGeneration(t *testing.T) {
	previousIntervals := awgLoopIntervals
	awgLoopIntervals = func() (time.Duration, time.Duration) { return time.Hour, time.Hour }
	t.Cleanup(func() { awgLoopIntervals = previousIntervals })

	a := NewApp()
	for generation := range 3 {
		a.startAWGLoops()
		a.awgMu.Lock()
		run := a.awgRun
		a.awgMu.Unlock()
		if run == nil {
			t.Fatalf("generation %d did not publish a loop", generation)
		}
		a.startAWGLoops()
		a.awgMu.Lock()
		if a.awgRun != run {
			t.Fatalf("generation %d duplicate Start replaced a live loop", generation)
		}
		a.awgMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		if err := a.stopAWGLoops(ctx); err != nil {
			cancel()
			t.Fatalf("generation %d stop: %v", generation, err)
		}
		cancel()
		select {
		case <-run.done:
		default:
			t.Fatalf("generation %d loop survived Stop", generation)
		}
	}
}
