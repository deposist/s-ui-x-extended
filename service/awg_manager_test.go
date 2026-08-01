package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeAWGProvisioner struct {
	snapshot func(context.Context) (AWGPeerSnapshot, error)
	add      func(context.Context, AWGPeerSpec) error
	remove   func(context.Context, string) error
}

func waitAWGManagerQueueLength(t *testing.T, manager *AWGManager, want int) {
	t.Helper()
	manager.mu.Lock()
	run := manager.run
	manager.mu.Unlock()
	if run == nil {
		t.Fatal("AWG manager has no active run")
	}
	deadline := time.Now().Add(time.Second)
	for len(run.commands) != want {
		if time.Now().After(deadline) {
			t.Fatalf("AWG manager queue length = %d, want %d", len(run.commands), want)
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *fakeAWGProvisioner) Snapshot(ctx context.Context) (AWGPeerSnapshot, error) {
	return f.snapshot(ctx)
}

func (f *fakeAWGProvisioner) Add(ctx context.Context, peer AWGPeerSpec) error {
	return f.add(ctx, peer)
}

func (f *fakeAWGProvisioner) Remove(ctx context.Context, publicKey string) error {
	return f.remove(ctx, publicKey)
}

func TestAWGManagerSerializesProvisionerCommands(t *testing.T) {
	var inFlight atomic.Int32
	var maximum atomic.Int32
	provisioner := &fakeAWGProvisioner{}
	record := func() {
		current := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	provisioner.snapshot = func(context.Context) (AWGPeerSnapshot, error) { record(); return nil, nil }
	provisioner.add = func(context.Context, AWGPeerSpec) error { record(); return nil }
	provisioner.remove = func(context.Context, string) error { record(); return nil }

	manager := NewAWGManager(nil, provisioner, 32)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); _, _ = manager.Snapshot(context.Background()) }()
		go func() { defer wg.Done(); _ = manager.Add(context.Background(), AWGPeerSpec{}) }()
		go func() { defer wg.Done(); _ = manager.Remove(context.Background(), "peer") }()
	}
	wg.Wait()
	if got := maximum.Load(); got != 1 {
		t.Fatalf("maximum concurrent provisioner calls = %d, want 1", got)
	}
}

func TestAWGManagerCanceledQueuedCommandDoesNotRun(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var removeCalls atomic.Int32
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) {
			close(started)
			<-release
			return nil, nil
		},
		add: func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error {
			removeCalls.Add(1)
			return nil
		},
	}
	manager := NewAWGManager(nil, provisioner, 1)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	firstDone := make(chan error, 1)
	go func() { _, err := manager.Snapshot(context.Background()); firstDone <- err }()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	queuedDone := make(chan error, 1)
	go func() { queuedDone <- manager.Remove(ctx, "peer") }()
	waitAWGManagerQueueLength(t, manager, 1)
	cancel()
	if err := <-queuedDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("queued command error = %v, want context.Canceled", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if got := removeCalls.Load(); got != 0 {
		t.Fatalf("canceled queued command ran %d times", got)
	}
}

func TestAWGManagerCallerCancellationInterruptsInFlightCommand(t *testing.T) {
	started := make(chan struct{})
	completed := make(chan struct{})
	provisioner := &fakeAWGProvisioner{
		snapshot: func(ctx context.Context) (AWGPeerSnapshot, error) {
			close(started)
			<-ctx.Done()
			close(completed)
			return nil, ctx.Err()
		},
		add:    func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error { return nil },
	}
	manager := NewAWGManager(nil, provisioner, 1)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	ctx, cancel := context.WithCancel(context.Background())
	commandDone := make(chan error, 1)
	go func() { _, err := manager.Snapshot(ctx); commandDone <- err }()
	<-started
	cancel()
	if err := <-commandDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight caller error = %v, want context.Canceled", err)
	}
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("caller cancellation did not reach the in-flight command")
	}
}

func TestAWGManagerBoundedQueueAppliesBackpressure(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var removeCalls atomic.Int32
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) {
			close(started)
			<-release
			return nil, nil
		},
		add: func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error {
			removeCalls.Add(1)
			return nil
		},
	}
	manager := NewAWGManager(nil, provisioner, 1)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Stop(context.Background()) })

	firstDone := make(chan error, 1)
	go func() { _, err := manager.Snapshot(context.Background()); firstDone <- err }()
	<-started
	secondDone := make(chan error, 1)
	go func() { secondDone <- manager.Remove(context.Background(), "second") }()
	waitAWGManagerQueueLength(t, manager, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := manager.Remove(ctx, "third"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("full queue error = %v, want context.DeadlineExceeded", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if err := <-secondDone; err != nil {
		t.Fatal(err)
	}
	if got := removeCalls.Load(); got != 1 {
		t.Fatalf("remove calls = %d, want only queued command", got)
	}
}

func TestAWGManagerStopCancelsInFlightAndRejectsQueue(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	var removeCalls atomic.Int32
	provisioner := &fakeAWGProvisioner{
		snapshot: func(ctx context.Context) (AWGPeerSnapshot, error) {
			close(started)
			<-ctx.Done()
			close(finished)
			return nil, ctx.Err()
		},
		add: func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error {
			removeCalls.Add(1)
			return nil
		},
	}
	manager := NewAWGManager(nil, provisioner, 1)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}

	firstDone := make(chan error, 1)
	go func() { _, err := manager.Snapshot(context.Background()); firstDone <- err }()
	<-started
	queuedDone := make(chan error, 1)
	go func() { queuedDone <- manager.Remove(context.Background(), "peer") }()
	waitAWGManagerQueueLength(t, manager, 1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	default:
		t.Fatal("Stop returned before the in-flight command observed cancellation")
	}
	if err := <-firstDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight command error = %v, want context.Canceled", err)
	}
	if err := <-queuedDone; !errors.Is(err, ErrAWGManagerStopped) {
		t.Fatalf("queued command error = %v, want ErrAWGManagerStopped", err)
	}
	if got := removeCalls.Load(); got != 0 {
		t.Fatalf("queued command ran %d times during shutdown", got)
	}
	if _, err := manager.Snapshot(context.Background()); !errors.Is(err, ErrAWGManagerStopped) {
		t.Fatalf("post-stop command error = %v, want ErrAWGManagerStopped", err)
	}
}

func TestAWGManagerCanRestartAfterStop(t *testing.T) {
	var calls atomic.Int32
	provisioner := &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) {
			calls.Add(1)
			return nil, nil
		},
		add:    func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error { return nil },
	}
	manager := NewAWGManager(nil, provisioner, 1)
	for i := 0; i < 2; i++ {
		if err := manager.Start(); err != nil {
			t.Fatal(err)
		}
		if err := manager.Start(); err != nil {
			t.Fatalf("idempotent Start: %v", err)
		}
		if _, err := manager.Snapshot(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := manager.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := manager.Stop(context.Background()); err != nil {
			t.Fatalf("idempotent Stop: %v", err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("snapshot calls = %d, want 2", got)
	}
}

func TestAWGManagerStopTimeoutStillCancelsInFlightCommand(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	provisioner := &fakeAWGProvisioner{
		snapshot: func(ctx context.Context) (AWGPeerSnapshot, error) {
			close(started)
			<-ctx.Done()
			close(finished)
			return nil, ctx.Err()
		},
		add:    func(context.Context, AWGPeerSpec) error { return nil },
		remove: func(context.Context, string) error { return nil },
	}
	manager := NewAWGManager(nil, provisioner, 1)
	if err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	commandDone := make(chan error, 1)
	go func() { _, err := manager.Snapshot(context.Background()); commandDone <- err }()
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := manager.Stop(ctx); err != nil {
		t.Fatalf("Stop error = %v", err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("Stop cancellation did not reach in-flight command")
	}
	if err := <-commandDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("command error = %v, want context.Canceled", err)
	}
}
