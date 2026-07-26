package core

import "fmt"

// WireGuardIPC exposes the live WireGuard userspace configuration interface.
type WireGuardIPC interface {
	IpcGet() (string, error)
	IpcSet(string) error
}

// WithWireGuardIPC runs fn against a live WireGuard endpoint while holding the
// core runtime read lock. This prevents the endpoint from being stopped during
// the IPC operation.
func (c *Core) WithWireGuardIPC(tag string, fn func(WireGuardIPC) error) error {
	if fn == nil {
		return fmt.Errorf("wireguard IPC callback is nil")
	}
	c.wireGuardIPCAccess.Lock()
	defer c.wireGuardIPCAccess.Unlock()
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
