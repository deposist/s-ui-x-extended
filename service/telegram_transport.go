package service

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/deposist/s-ui-x/util/common"

	M "github.com/sagernet/sing/common/metadata"
)

// newCoreOutboundHTTPClient builds an HTTP client whose TCP connections are
// dialed through a running sing-box outbound (by tag). This lets the Telegram
// modules egress through a configured proxy/VPN outbound instead of (or in
// addition to) an HTTP/SOCKS proxy. Requires the core to be running and the
// outbound to exist; otherwise returns an error so the caller can surface it.
func newCoreOutboundHTTPClient(tag string, timeout time.Duration) (*http.Client, error) {
	if tag == "" {
		return nil, common.NewError("outbound tag is empty")
	}
	coreInstance := DefaultRuntime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil, common.NewError("core is not running; cannot use outbound transport")
	}
	manager := coreInstance.OutboundManager()
	if manager == nil {
		return nil, common.NewError("core outbound manager unavailable")
	}
	ob, ok := manager.Outbound(tag)
	if !ok {
		return nil, common.NewErrorf("outbound not found: %s", tag)
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return ob.DialContext(ctx, network, M.ParseSocksaddr(addr))
		},
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &http.Client{Timeout: timeout, Transport: transport}, nil
}
