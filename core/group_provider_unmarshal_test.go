package core

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"
)

// TestGroupAndProviderConfigsUnmarshal verifies that the panel-assembled group
// and provider configurations unmarshal through the linked core registries
// without build tags. This covers selector (with providers), urltest, fallback,
// and the failover-assembled-as-selector pattern, plus inline/local/remote
// provider definitions.
func TestGroupAndProviderConfigsUnmarshal(t *testing.T) {
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry())

	configs := map[string][]byte{
		"selector_with_providers": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"socks","tag":"m1","server":"127.0.0.1","server_port":1080},
				{"type":"selector","tag":"sel","outbounds":["m1","direct"],"providers":["prov-inline"],"default":"m1"}
			],
			"providers":[
				{"type":"inline","tag":"prov-inline","outbounds":[{"type":"socks","tag":"prov-m","server":"127.0.0.1","server_port":1081}]}
			],
			"route":{"rules":[]}
		}`),
		"urltest_basic": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"socks","tag":"m1","server":"127.0.0.1","server_port":1080},
				{"type":"urltest","tag":"ut","outbounds":["m1"],"url":"https://www.gstatic.com/generate_204","interval":"1m","tolerance":50}
			],
			"route":{"rules":[]}
		}`),
		"fallback_basic": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"socks","tag":"m1","server":"127.0.0.1","server_port":1080},
				{"type":"socks","tag":"m2","server":"127.0.0.1","server_port":1081},
				{"type":"fallback","tag":"fb","outbounds":["m1","m2"]}
			],
			"route":{"rules":[]}
		}`),
		"failover_assembled_as_selector": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"socks","tag":"us-1","server":"127.0.0.1","server_port":1080},
				{"type":"socks","tag":"us-2","server":"127.0.0.1","server_port":1081},
				{"type":"selector","tag":"auto-us","outbounds":["us-1","us-2"],"default":"us-1","interrupt_exist_connections":true}
			],
			"route":{"rules":[]}
		}`),
		"selector_use_all_providers": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"selector","tag":"all-prov","outbounds":["direct"],"use_all_providers":true}
			],
			"providers":[
				{"type":"inline","tag":"p1","outbounds":[{"type":"socks","tag":"pm1","server":"127.0.0.1","server_port":1080}]}
			],
			"route":{"rules":[]}
		}`),
		"selector_with_include_exclude": []byte(`{
			"log":{"disabled":true},
			"dns":{"servers":[],"rules":[]},
			"outbounds":[
				{"type":"direct","tag":"direct"},
				{"type":"selector","tag":"filtered","outbounds":["direct"],"providers":["p1"],"include":"us-.*","exclude":"bad-.*"}
			],
			"providers":[
				{"type":"inline","tag":"p1","outbounds":[{"type":"socks","tag":"us-1","server":"127.0.0.1","server_port":1080}]}
			],
			"route":{"rules":[]}
		}`),
	}

	for name, config := range configs {
		t.Run(name, func(t *testing.T) {
			var options option.Options
			if err := options.UnmarshalJSONContext(ctx, config); err != nil {
				t.Fatalf("unmarshal %s: %v", name, err)
			}
		})
	}
}
