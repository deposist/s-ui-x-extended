package network

import (
	"net"

	"github.com/deposist/s-ui-x-extended/logger"
)

type ListenResult struct {
	Listener      net.Listener
	RequestedAddr string
	FallbackAddr  string
	Fallback      bool
	BindError     error
}

// ListenWithFallback opens a TCP listener on `addr`. When `host` is a literal
// IP that the host kernel cannot bind (the typical "EADDRNOTAVAIL" error
// after restoring a backup from another machine), the function logs a
// warning and retries on the loopback interface (127.0.0.1) only. This keeps
// the panel reachable locally (e.g. over an SSH tunnel) so the operator can
// correct the listen address from the UI, without ever silently widening an
// intentionally-restricted bind to every (public) interface.
//
// `host` is the bare host portion (no port) used by the caller to build
// `addr`; pass an empty string when the address is already an "any"
// address or when no fallback is desired.
func ListenWithFallback(addr, host string, port string) (net.Listener, error) {
	result, err := ListenWithFallbackResult(addr, host, port)
	if err != nil {
		return nil, err
	}
	return result.Listener, nil
}

func ListenWithFallbackResult(addr, host string, port string) (ListenResult, error) {
	result := ListenResult{RequestedAddr: addr}
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		result.Listener = listener
		return result, nil
	}
	if !shouldFallback(err) || host == "" {
		return result, err
	}
	// Fall back to loopback only, never the "" wildcard (0.0.0.0/::): an operator
	// who deliberately narrowed the bind must not be silently re-exposed on every
	// public interface by a stale/unbindable address. Loopback keeps the panel
	// reachable for local recovery (SSH tunnel) so the address can be corrected.
	fallback := net.JoinHostPort("127.0.0.1", port)
	logger.Warningf(
		"could not bind on %s (%v); falling back to loopback %s. Update the listen address from the UI to silence this warning.",
		addr, err, fallback,
	)
	listener, fallbackErr := net.Listen("tcp", fallback)
	if fallbackErr != nil {
		return result, fallbackErr
	}
	result.Listener = listener
	result.FallbackAddr = fallback
	result.Fallback = true
	result.BindError = err
	return result, nil
}

// shouldFallback reports whether err is the kind of bind error that points
// at a stale listen address inherited from another machine (the address is
// syntactically valid but the kernel does not own it).
func shouldFallback(err error) bool {
	return isAddrNotAvailable(err)
}
