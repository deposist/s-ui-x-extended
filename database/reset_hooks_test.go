package database

import (
	"context"
	"errors"
	"testing"
)

func TestResetCachesReturnsHookError(t *testing.T) {
	const hookName = "test.reset_caches_error"
	expected := errors.New("reset hook failed")
	calls := 0
	RegisterResetHook(hookName, func() error {
		calls++
		return expected
	})
	t.Cleanup(func() { RegisterResetHook(hookName, nil) })

	err := ResetCaches(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("ResetCaches error = %v, want %v", err, expected)
	}
	if calls != 1 {
		t.Fatalf("reset hook calls = %d, want 1", calls)
	}
}

func TestResetCachesRunsLaterHooksAfterError(t *testing.T) {
	firstName := "test.reset_caches_error_first"
	secondName := "test.reset_caches_error_second"
	firstErr := errors.New("first reset hook failed")
	secondErr := errors.New("second reset hook failed")
	secondCalls := 0
	RegisterResetHook(firstName, func() error { return firstErr })
	RegisterResetHook(secondName, func() error {
		secondCalls++
		return secondErr
	})
	t.Cleanup(func() {
		RegisterResetHook(firstName, nil)
		RegisterResetHook(secondName, nil)
	})

	err := ResetCaches(context.Background())
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("ResetCaches error = %v, want both hook errors", err)
	}
	if secondCalls != 1 {
		t.Fatalf("later reset hook calls = %d, want 1", secondCalls)
	}
}
