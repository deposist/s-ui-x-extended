package app

import (
	"errors"
	"reflect"
	"testing"
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
