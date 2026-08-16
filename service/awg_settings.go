package service

import (
	"net/netip"
)

// AWGSettings contains the validated settings used by a managed endpoint's
// device manager. Endpoint-scoped managers build it from endpoint Ext metadata
// via AWGSettingsForEndpoint.
type AWGSettings struct {
	Enabled              bool
	EndpointTag          string
	PublicEndpoint       string
	Subnet               netip.Prefix
	DNS                  []netip.Addr
	DefaultDeviceLimit   int
	ReconcileIntervalSec int
	StatsIntervalSec     int
	MTU                  uint32
	// ClientAllowedIPs / ClientKeepalive override the AllowedIPs and
	// PersistentKeepalive lines in rendered device configs. Zero values keep
	// the historical defaults (0.0.0.0/0, ::/0 and 25).
	ClientAllowedIPs []string
	ClientKeepalive  int
}
