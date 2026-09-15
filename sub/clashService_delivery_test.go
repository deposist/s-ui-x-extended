package sub

import (
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core/capabilities"

	"gopkg.in/yaml.v3"
)

// minimalClashOutbound is the smallest outbound the converter can render: the
// per-protocol fields are absent on purpose, because this test is about which
// types reach the proxies list at all.
func minimalClashOutbound(typ string) map[string]interface{} {
	return map[string]interface{}{
		"type":        typ,
		"tag":         typ + "-node",
		"server":      "example.com",
		"server_port": 443,
	}
}

func clashProxyNames(t *testing.T, rendered string) map[string]bool {
	t.Helper()
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(rendered), &config); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	names := map[string]bool{}
	for _, p := range config["proxies"].([]interface{}) {
		if pm, ok := p.(map[string]interface{}); ok {
			if name, ok := pm["name"].(string); ok {
				names[name] = true
			}
		}
	}
	return names
}

// TestConvertToClashMetaImplementsEveryManifestProxyType ties the capability
// manifest to the converter: the manifest is the single source of truth for what
// a Clash subscription may carry, so a type marked clashDelivery=proxy must
// actually reach the proxies list. Without this, marking a protocol as
// clash-representable (or adding its mapping) could quietly do nothing.
func TestConvertToClashMetaImplementsEveryManifestProxyType(t *testing.T) {
	var outbounds []map[string]interface{}
	var expected []string
	for typ := range capabilities.ClashProxyTypes() {
		outbounds = append(outbounds, minimalClashOutbound(typ))
		expected = append(expected, typ+"-node")
	}
	rendered, err := (&ClashService{}).ConvertToClashMeta(&outbounds, basicClashConfig)
	if err != nil {
		t.Fatalf("ConvertToClashMeta: %v", err)
	}
	names := clashProxyNames(t, rendered)
	for _, name := range expected {
		if !names[name] {
			t.Errorf("manifest declares %q clash-representable but the converter dropped it", name)
		}
	}
}

// TestConvertToClashMetaOmitsUnrepresentableWithReason is the other half of the
// contract: a protocol the format cannot carry must not be emitted as a proxy
// (mihomo would either reject the config or dial a node without its secret) and
// must not vanish silently either - the rendered config names the omission.
func TestConvertToClashMetaOmitsUnrepresentableWithReason(t *testing.T) {
	unsupported := capabilities.ClashUnsupportedTypes()
	if len(unsupported) == 0 {
		t.Fatal("manifest declares no clash-unsupported outbound types; the gate in ConvertToClashMeta can no longer be exercised")
	}
	outbounds := []map[string]interface{}{minimalClashOutbound("vless")}
	for _, typ := range unsupported {
		outbounds = append(outbounds, minimalClashOutbound(typ))
	}

	rendered, err := (&ClashService{}).ConvertToClashMeta(&outbounds, basicClashConfig)
	if err != nil {
		t.Fatalf("ConvertToClashMeta: %v", err)
	}
	names := clashProxyNames(t, rendered)
	if !names["vless-node"] {
		t.Error("a representable proxy was dropped alongside the unsupported ones")
	}
	for _, typ := range unsupported {
		if names[typ+"-node"] {
			t.Errorf("unsupported type %q was rendered as a clash proxy", typ)
		}
	}
	for _, typ := range unsupported {
		if !strings.Contains(rendered, typ+"-node ("+typ+")") {
			t.Errorf("rendered config does not explain the omitted node of type %q", typ)
		}
	}
}

// TestClashDeliveryManifestIsConsistent guards the manifest classification from
// contradiction: a type may not be both a clash proxy and clash-unsupported.
func TestClashDeliveryManifestIsConsistent(t *testing.T) {
	proxy := capabilities.ClashProxyTypes()
	for _, typ := range capabilities.ClashUnsupportedTypes() {
		if _, both := proxy[typ]; both {
			t.Errorf("outbound %q is classified as both proxy and unsupported", typ)
		}
	}
	for _, out := range capabilities.Outbounds() {
		if _, ok := proxy[out.Type]; ok && out.ClashDelivery != "proxy" {
			t.Errorf("outbound %q is in the proxy set but declares clashDelivery=%q", out.Type, out.ClashDelivery)
		}
	}
}