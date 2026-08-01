package core

import (
	"context"
	"fmt"
)

// WireGuardIPC exposes the live WireGuard userspace configuration interface.
type WireGuardIPC interface {
	IpcGet() (string, error)
	IpcSet(string) error
}

// WithWireGuardIPC runs fn against a live WireGuard endpoint while holding the
// core runtime read lock. This prevents the endpoint from being stopped during
// the IPC operation.
func (c *Core) WithWireGuardIPC(tag string, fn func(WireGuardIPC) error) error {
	return c.WithWireGuardIPCContext(context.Background(), tag, fn)
}

// WithWireGuardIPCContext is cancellation-aware while waiting to enter the IPC
// critical section. The third-party UAPI itself has no context API, so once fn
// begins it remains atomic with respect to Core Start/Stop and must return from
// the underlying finite transport operation before the lock can be released.
func (c *Core) WithWireGuardIPCContext(ctx context.Context, tag string, fn func(WireGuardIPC) error) error {
	if fn == nil {
		return fmt.Errorf("wireguard IPC callback is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.wireGuardIPCAccess:
	}
	defer func() { c.wireGuardIPCAccess <- struct{}{} }()
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.withRuntime(func(rt coreRuntime) error {
		endpoint, ok := rt.endpointManager.Get(tag)
		if !ok {
			return fmt.Errorf("wireguard endpoint %q not found", tag)
		}
		ipc, ok := endpoint.(WireGuardIPC)
		if !ok {
			return fmt.Errorf("endpoint %q does not expose WireGuard UAPI", tag)
		}
		return fn(ipc)
	})
}
