package service

import (
	"net/netip"
)

// managedTestAWGSettings mirrors the endpoint-derived settings of a managed
// endpoint with the 10.77.0.0/16 defaults.
func managedTestAWGSettings() AWGSettings {
	return AWGSettings{
		Enabled:              true,
		EndpointTag:          "managed-awg",
		PublicEndpoint:       "vpn.example.com:51820",
		Subnet:               netip.MustParsePrefix("10.77.0.0/16"),
		DNS:                  []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("1.0.0.1")},
		DefaultDeviceLimit:   3,
		ReconcileIntervalSec: 30,
		StatsIntervalSec:     60,
	}
}
