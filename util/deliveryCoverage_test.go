package util

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
)

// TestManifestDeliveryIsImplemented fail-closes the client-delivery contract:
// the capability manifest may only claim a delivery channel this package
// actually implements. A protocol added to the manifest without a generator
// (link or JSON outbound) fails here instead of silently disappearing from
// every subscription, which is the failure mode an operator cannot see.
func TestManifestDeliveryIsImplemented(t *testing.T) {
	cfg := json.RawMessage(wellFormedClientConfig)
	for _, in := range capabilities.Inbounds() {
		if in.Alias {
			continue
		}
		switch in.ClientDelivery {
		case "uri", "telegram":
			t.Run("link/"+in.Type, func(t *testing.T) {
				prefixes, declared := linkSchemePrefixes[in.Type]
				if !declared {
					t.Fatalf("%s claims clientDelivery=%s but the test does not declare the URI scheme(s) LinkGenerator must emit for it; declare them (and the generator) before shipping the protocol", in.Type, in.ClientDelivery)
				}
				links := LinkGenerator(cfg, wellFormedInbound(in.Type), "example.com")
				if len(links) == 0 {
					t.Fatalf("%s claims clientDelivery=%s but LinkGenerator produced no link", in.Type, in.ClientDelivery)
				}
				for _, link := range links {
					if !hasAnyPrefix(link, prefixes) {
						t.Fatalf("%s produced link %q, want one of %v", in.Type, link, prefixes)
					}
				}
			})
		case "json":
			t.Run("json/"+in.Type, func(t *testing.T) {
				key := capabilities.OutJSONBuilders()[in.Type]
				if key == "" {
					t.Fatalf("%s claims clientDelivery=json without an outJsonBuilder key", in.Type)
				}
				if _, ok := outJSONBuilders[key]; !ok {
					t.Fatalf("%s maps to outJsonBuilder %q which has no builder in the FillOutJson dispatch table", in.Type, key)
				}
			})
		case "none", "broken":
			t.Run("none/"+in.Type, func(t *testing.T) {
				if links := LinkGenerator(cfg, wellFormedInbound(in.Type), "example.com"); len(links) != 0 {
					t.Fatalf("%s claims clientDelivery=%s but LinkGenerator produced %d link(s)", in.Type, in.ClientDelivery, len(links))
				}
			})
		}
	}
}

// TestOutJSONBuildersAreReachableEverywhere guards the dispatch table from the
// other direction: a builder key that no manifest row references (a typo or a
// leftover) is dead code, and a manifest key that is misspelled has no builder.
func TestOutJSONBuildersAreReachableEverywhere(t *testing.T) {
	referenced := map[string]struct{}{}
	for _, in := range capabilities.Inbounds() {
		if in.OutJSONBuilder != "" {
			referenced[in.OutJSONBuilder] = struct{}{}
		}
	}
	for key := range outJSONBuilders {
		if key == "base" {
			// "base" is the deliberate no-op builder: the manifest leaves the key
			// empty for the protocols that need no protocol-specific fields.
			continue
		}
		if _, ok := referenced[key]; !ok {
			t.Errorf("outJsonBuilder %q is implemented but referenced by no manifest entry", key)
		}
	}
}