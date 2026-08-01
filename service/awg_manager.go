package service

import (
	"context"
	"errors"
	"sync"
)

const defaultAWGManagerQueueCapacity = 64

var (
	ErrAWGManagerStopped     = errors.New("AWG manager is stopped")
	ErrAWGManagerStopping    = errors.New("AWG manager is stopping")
	ErrAWGProvisionerMissing = errors.New("AWG provisioner is not configured")
)

type awgCommandResult struct {
	snapshot AWGPeerSnapshot
	device   AWGDeviceInfo
	value    any
	err      error
}

type awgCommand struct {
	ctx    context.Context
	run    func(context.Context) awgCommandResult
	result chan awgCommandResult
}

type awgManagerRun struct {
	commands chan awgCommand
	ctx      context.Context
	cancel   context.CancelFunc
	stopCh   chan struct{}
	done     chan struct{}
	stopOnce sync.Once
	submitWG sync.WaitGroup
}

// AWGManager is the single entry point for live managed WireGuard operations.
// Database-only reads may bypass it, but snapshots and all UAPI mutations must
// be submitted to this worker.
type AWGManager struct {
	runtime     *Runtime
	provisioner AWGProvisioner
	capacity    int
	deps        AWGManagerDeps

	mu        sync.Mutex
	run       *awgManagerRun
	accepting bool
}

func NewAWGManager(runtime *Runtime, provisioner AWGProvisioner, capacity int) *AWGManager {
	return NewAWGManagerWithDeps(runtime, provisioner, capacity, defaultAWGManagerDeps())
}

func NewAWGManagerWithDeps(runtime *Runtime, provisioner AWGProvisioner, capacity int, deps AWGManagerDeps) *AWGManager {
	if capacity <= 0 {
		capacity = defaultAWGManagerQueueCapacity
	}
	return &AWGManager{
		runtime:     runtimeOrDefault(runtime),
		provisioner: provisioner,
		capacity:    capacity,
		deps:        deps,
	}
}

func (m *AWGManager) Start() error {
	if m == nil {
		return ErrAWGManagerStopped
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.accepting {
		return nil
	}
	if m.run != nil {
		select {
		case <-m.run.done:
		default:
			return ErrAWGManagerStopping
		}
	}
	runCtx, cancel := context.WithCancel(context.Background())
	run := &awgManagerRun{
		commands: make(chan awgCommand, m.capacity),
		stopCh:   make(chan struct{}),
		done:     make(chan struct{}),
		ctx:      runCtx,
		cancel:   cancel,
	}
	m.run = run
	m.accepting = true
	go m.worker(run)
	return nil
}

func (m *AWGManager) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	run := m.run
	if run == nil {
		m.mu.Unlock()
		return nil
	}
	m.accepting = false
	run.cancel()
	run.stopOnce.Do(func() { close(run.stopCh) })
	m.mu.Unlock()

	select {
	case <-run.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *AWGManager) Snapshot(ctx context.Context) (AWGPeerSnapshot, error) {
	result := m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		if m.provisioner == nil {
			return awgCommandResult{err: ErrAWGProvisionerMissing}
		}
		snapshot, err := m.provisioner.Snapshot(operationCtx)
		return awgCommandResult{snapshot: snapshot, err: err}
	})
	return result.snapshot, result.err
}

func (m *AWGManager) Add(ctx context.Context, peer AWGPeerSpec) error {
	return m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		if m.provisioner == nil {
			return awgCommandResult{err: ErrAWGProvisionerMissing}
		}
		return awgCommandResult{err: m.provisioner.Add(operationCtx, peer)}
	}).err
}

func (m *AWGManager) Remove(ctx context.Context, publicKey string) error {
	return m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		if m.provisioner == nil {
			return awgCommandResult{err: ErrAWGProvisionerMissing}
		}
		return awgCommandResult{err: m.provisioner.Remove(operationCtx, publicKey)}
	}).err
}

func (m *AWGManager) submit(ctx context.Context, operation func(context.Context) awgCommandResult) awgCommandResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return awgCommandResult{err: err}
	}
	if m == nil {
		return awgCommandResult{err: ErrAWGManagerStopped}
	}

	m.mu.Lock()
	run := m.run
	if !m.accepting || run == nil {
		m.mu.Unlock()
		return awgCommandResult{err: ErrAWGManagerStopped}
	}
	run.submitWG.Add(1)
	m.mu.Unlock()

	command := awgCommand{
		ctx:    ctx,
		run:    operation,
		result: make(chan awgCommandResult, 1),
	}
	select {
	case run.commands <- command:
	case <-ctx.Done():
		run.submitWG.Done()
		return awgCommandResult{err: ctx.Err()}
	case <-run.stopCh:
		run.submitWG.Done()
		return awgCommandResult{err: ErrAWGManagerStopped}
	}
	run.submitWG.Done()

	select {
	case result := <-command.result:
		return result
	case <-ctx.Done():
		return awgCommandResult{err: ctx.Err()}
	}
}

func (m *AWGManager) worker(run *awgManagerRun) {
	defer close(run.done)
	for {
		select {
		case <-run.stopCh:
			m.rejectQueued(run)
			return
		default:
		}

		select {
		case <-run.stopCh:
			m.rejectQueued(run)
			return
		case command := <-run.commands:
			select {
			case <-run.stopCh:
				command.result <- awgCommandResult{err: ErrAWGManagerStopped}
				m.rejectQueued(run)
				return
			default:
			}
			if err := command.ctx.Err(); err != nil {
				command.result <- awgCommandResult{err: err}
				continue
			}
			operationCtx, cancel := context.WithCancelCause(command.ctx)
			stopRunCancellation := context.AfterFunc(run.ctx, func() {
				cancel(context.Cause(run.ctx))
			})
			result := command.run(operationCtx)
			stopRunCancellation()
			cancel(nil)
			command.result <- result
		}
	}
}

func (m *AWGManager) rejectQueued(run *awgManagerRun) {
	// Stop first disables admission while holding m.mu. Therefore submitWG can
	// safely reach zero before the final drain without racing a later Add.
	run.submitWG.Wait()
	for {
		select {
		case command := <-run.commands:
			command.result <- awgCommandResult{err: ErrAWGManagerStopped}
		default:
			return
		}
	}
}
