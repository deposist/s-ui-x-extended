package service

import (
	"sync"
	"testing"
)

func TestRuntimeCoreStartCooldownConcurrentAccessRaceAnchorIssue20(t *testing.T) {
	r := NewRuntimeWithCoreProvider(nil)

	const (
		goroutines = 64
		iterations = 1000
	)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				if worker%2 == 0 {
					r.markCoreStartFailed()
					r.markCoreStartSucceeded()
					continue
				}
				_ = r.startCooldownActive()
				_ = r.coreStartCooldownDuration()
			}
		}(i)
	}
	close(start)
	wg.Wait()
}

func TestRuntimeCoreStartCooldownActiveAfterFailureIssue20(t *testing.T) {
	r := NewRuntimeWithCoreProvider(nil)

	r.markCoreStartFailed()
	if !r.startCooldownActive() {
		t.Fatal("core start cooldown should be active after a failure")
	}

	r.markCoreStartSucceeded()
	if r.startCooldownActive() {
		t.Fatal("core start cooldown should clear after a successful start")
	}
}
