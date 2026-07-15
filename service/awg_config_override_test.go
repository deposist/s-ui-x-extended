package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func newAWGOverrideTestDevice(t *testing.T, overrides AWGSettings) (*AWGManager, uint, uint) {
	t.Helper()
	manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
		snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
		add:      func(context.Context, AWGPeerSpec) error { return nil },
		remove:   func(context.Context, string) error { return nil },
	})
	serverPrivate := wgtypes.Key(bytes.Repeat([]byte{0x55}, 32))
	options, err := json.Marshal(map[string]any{
		"address": []string{"10.78.0.1/29"}, "private_key": serverPrivate.String(), "listen_port": 51821, "mtu": 1380,
		"amnezia": map[string]any{"jc": 3, "jmin": 10, "jmax": 20, "s1": 15, "s2": 18, "s3": 12, "s4": 8,
			"h1": "1000-1099", "h2": 2000, "h3": 3000, "h4": 4000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Endpoint{Type: "wireguard", Tag: "awg-override", Options: options}).Error; err != nil {
		t.Fatal(err)
	}
	manager.deps.LoadSettings = func() (AWGSettings, error) {
		settings := AWGSettings{
			Enabled: true, EndpointTag: "awg-override", PublicEndpoint: "vpn.example.com:51821",
			Subnet: netip.MustParsePrefix("10.78.0.0/29"), DNS: []netip.Addr{netip.MustParseAddr("1.1.1.1")},
			ClientAllowedIPs: overrides.ClientAllowedIPs, ClientKeepalive: overrides.ClientKeepalive,
		}
		return settings, nil
	}
	client := createAWGEligibleClient(t)
	device, err := manager.CreateDevice(context.Background(), client.Id, "override-config", "Phone", 1)
	if err != nil {
		t.Fatal(err)
	}
	return manager, device.ID, client.Id
}

func renderAWGOverrideConfig(t *testing.T, overrides AWGSettings) string {
	t.Helper()
	manager, deviceID, clientID := newAWGOverrideTestDevice(t, overrides)
	config, err := manager.RenderOwnedConfig(context.Background(), deviceID, clientID)
	if err != nil {
		t.Fatal(err)
	}
	return string(config)
}

func TestRenderOwnedConfigDefaultAllowedIPsAndKeepalive(t *testing.T) {
	config := renderAWGOverrideConfig(t, AWGSettings{})
	if !strings.Contains(config, "AllowedIPs = 0.0.0.0/0, ::/0\n") {
		t.Fatalf("default AllowedIPs missing:\n%s", config)
	}
	if !strings.Contains(config, "PersistentKeepalive = 25\n") {
		t.Fatalf("default keepalive missing:\n%s", config)
	}
}

func TestRenderOwnedConfigAllowedIPsOverride(t *testing.T) {
	config := renderAWGOverrideConfig(t, AWGSettings{
		ClientAllowedIPs: []string{"10.78.0.0/29", "192.168.100.0/24"},
		ClientKeepalive:  55,
	})
	if !strings.Contains(config, "AllowedIPs = 10.78.0.0/29, 192.168.100.0/24\n") {
		t.Fatalf("AllowedIPs override missing:\n%s", config)
	}
	if !strings.Contains(config, "PersistentKeepalive = 55\n") {
		t.Fatalf("keepalive override missing:\n%s", config)
	}
	if strings.Contains(config, "0.0.0.0/0") {
		t.Fatalf("default AllowedIPs leaked alongside override:\n%s", config)
	}
}

// Defense in depth: even if invalid values somehow reach the renderer
// (metadata validation is the primary gate), they must never enter the INI
// text as directives.
func TestAWGClientAllowedIPsRendererDropsUnsafeValues(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"empty falls back to default", nil, "0.0.0.0/0, ::/0"},
		{"blank entries fall back", []string{"", "  "}, "0.0.0.0/0, ::/0"},
		{"newline injection dropped", []string{"10.0.0.0/8\nPersistentKeepalive = 1"}, "0.0.0.0/0, ::/0"},
		{"inner carriage return dropped", []string{"10.0.0.0/8\rX"}, "0.0.0.0/0, ::/0"},
		// A trailing \r is removed by TrimSpace, leaving a plain valid prefix
		// with nothing left to inject.
		{"trailing carriage return trimmed to safe value", []string{"10.0.0.0/8\r"}, "10.0.0.0/8"},
		{"non-CIDR dropped", []string{"not-a-prefix"}, "0.0.0.0/0, ::/0"},
		{"bare IP without prefix dropped", []string{"10.1.2.3"}, "0.0.0.0/0, ::/0"},
		{"valid survive invalid", []string{"10.0.0.0/8", "garbage"}, "10.0.0.0/8"},
		{"ipv6 prefix ok", []string{"2001:db8::/32"}, "2001:db8::/32"},
		{"whitespace trimmed", []string{" 10.0.0.0/8 "}, "10.0.0.0/8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := awgClientAllowedIPs(tc.in); got != tc.want {
				t.Fatalf("awgClientAllowedIPs(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestAWGClientKeepaliveBounds(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, 25}, {-1, 25}, {3601, 25}, {1, 1}, {25, 25}, {3600, 3600},
	}
	for _, tc := range cases {
		if got := awgClientKeepalive(tc.in); got != tc.want {
			t.Fatalf("awgClientKeepalive(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestValidateAWGEndpointMetadataClientOverrides(t *testing.T) {
	base := func(mutate func(*model.AWGEndpointMetadata)) model.Endpoint {
		metadata := model.AWGEndpointMetadata{
			Managed: true, PublicEndpoint: "vpn.example.com:51820",
			DNS: []string{"1.1.1.1"}, DefaultDeviceLimit: 3,
		}
		mutate(&metadata)
		ext, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		return model.Endpoint{Type: "wireguard", Tag: "meta-check", Ext: ext}
	}
	cases := []struct {
		name     string
		mutate   func(*model.AWGEndpointMetadata)
		errChunk string
	}{
		{"no overrides ok", func(m *model.AWGEndpointMetadata) {}, ""},
		{"valid cidrs ok", func(m *model.AWGEndpointMetadata) {
			m.ClientAllowedIPs = []string{"10.0.0.0/8", "2001:db8::/32"}
			m.ClientKeepalive = 30
		}, ""},
		{"invalid cidr rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientAllowedIPs = []string{"10.0.0.0/8", "not-a-cidr"}
		}, "invalid AWG client AllowedIPs"},
		{"bare ip rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientAllowedIPs = []string{"10.1.2.3"}
		}, "invalid AWG client AllowedIPs"},
		{"newline injection rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientAllowedIPs = []string{"10.0.0.0/8\nPersistentKeepalive = 1"}
		}, "invalid AWG client AllowedIPs"},
		{"empty entry rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientAllowedIPs = []string{"   "}
		}, "invalid AWG client AllowedIPs"},
		{"keepalive above bound rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientKeepalive = 3601
		}, "keepalive"},
		{"negative keepalive rejected", func(m *model.AWGEndpointMetadata) {
			m.ClientKeepalive = -1
		}, "keepalive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateAWGEndpointMetadata(base(tc.mutate))
			if tc.errChunk == "" {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.errChunk) {
				t.Fatalf("error %v does not contain %q", err, tc.errChunk)
			}
		})
	}
}

// Old Ext payloads without the new fields must keep parsing with zero values
// (backward compatibility: no migration, JSON metadata only).
func TestAWGEndpointMetadataBackwardCompatible(t *testing.T) {
	legacy := json.RawMessage(`{"managed":true,"publicEndpoint":"vpn.example.com:51820","dns":["1.1.1.1"],"defaultDeviceLimit":3}`)
	metadata, err := validateAWGEndpointMetadata(model.Endpoint{Type: "wireguard", Tag: "legacy", Ext: legacy})
	if err != nil {
		t.Fatalf("legacy metadata rejected: %v", err)
	}
	if len(metadata.ClientAllowedIPs) != 0 || metadata.ClientKeepalive != 0 {
		t.Fatalf("legacy metadata produced non-zero overrides: %+v", metadata)
	}
}
