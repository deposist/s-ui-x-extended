package importxui

import (
	"net"
	"strings"
	"testing"
)

func findServerByType(servers []any, typ string) map[string]any {
	for _, s := range servers {
		if m, ok := s.(map[string]any); ok && m["type"] == typ {
			return m
		}
	}
	return nil
}

func TestMapXrayDNS_ServersRulesStrategy(t *testing.T) {
	raw := map[string]any{
		"servers": []any{
			"8.8.8.8",
			"https://dns.google/dns-query",
			"tls://1.1.1.1",
			"localhost",
			map[string]any{"address": "8.8.4.4", "domains": []any{"geosite:google", "domain:example.com"}},
		},
		"queryStrategy": "UseIPv4",
		"hosts":         map[string]any{"a.com": "1.2.3.4"},
	}
	ruleSets := []any{}
	seen := map[string]struct{}{}
	out, warnings := mapXrayDNS(raw, &ruleSets, seen)

	servers, _ := out["servers"].([]any)
	if len(servers) != 5 {
		t.Fatalf("servers = %d, want 5: %#v", len(servers), servers)
	}
	if s0 := servers[0].(map[string]any); s0["type"] != "udp" || s0["server"] != "8.8.8.8" {
		t.Errorf("first server = %v, want udp/8.8.8.8", s0)
	} else if out["final"] != s0["tag"] {
		t.Errorf("final = %v, want %v", out["final"], s0["tag"])
	}
	if https := findServerByType(servers, "https"); https == nil || https["server"] != "dns.google" || https["path"] != "/dns-query" {
		t.Errorf("https server = %v", https)
	}
	if tlsS := findServerByType(servers, "tls"); tlsS == nil || tlsS["server"] != "1.1.1.1" {
		t.Errorf("tls server = %v", tlsS)
	}
	if findServerByType(servers, "local") == nil {
		t.Errorf("missing local server in %#v", servers)
	}
	if out["strategy"] != "ipv4_only" {
		t.Errorf("strategy = %v, want ipv4_only", out["strategy"])
	}

	rules, _ := out["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("dns rules = %#v, want 1", rules)
	}
	rule := rules[0].(map[string]any)
	if rule["server"] == nil {
		t.Errorf("dns rule missing server: %v", rule)
	}
	if rs, _ := rule["rule_set"].([]string); len(rs) != 1 || rs[0] != "geosite-google" {
		t.Errorf("dns rule rule_set = %v", rule["rule_set"])
	}
	if ds, _ := rule["domain_suffix"].([]string); len(ds) != 1 || ds[0] != "example.com" {
		t.Errorf("dns rule domain_suffix = %v", rule["domain_suffix"])
	}
	if len(ruleSets) != 1 {
		t.Errorf("geosite rule set should be registered once, got %#v", ruleSets)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "host override") {
		t.Errorf("expected a hosts warning, got %v", warnings)
	}
}

func TestMapXrayDNS_DomainServerGetsResolver(t *testing.T) {
	// A domain-addressed DNS server must get a domain_resolver pointing at an
	// IP-addressed bootstrap, or sing-box refuses to start ("missing domain
	// resolver for domain server address"). With an IP server present (1.1.1.1),
	// it is reused as the bootstrap — no extra server is added — and the s-ui tls
	// block is attached, matching a natively-created server.
	raw := map[string]any{"servers": []any{"1.1.1.1", "https://dns.google/dns-query", "tls://dns.adguard.com"}}
	ruleSets := []any{}
	seen := map[string]struct{}{}
	out, _ := mapXrayDNS(raw, &ruleSets, seen)
	servers := out["servers"].([]any)
	if len(servers) != 3 {
		t.Fatalf("servers = %d, want 3 (IP server reused as bootstrap, none added): %#v", len(servers), servers)
	}
	ipS := findServerByType(servers, "udp")
	if ipS == nil || ipS["server"] != "1.1.1.1" {
		t.Fatalf("expected udp 1.1.1.1 server, got %#v", servers)
	}
	if _, has := ipS["domain_resolver"]; has {
		t.Errorf("the IP bootstrap must not get a domain_resolver: %#v", ipS)
	}
	for _, typ := range []string{"https", "tls"} {
		srv := findServerByType(servers, typ)
		if srv == nil {
			t.Fatalf("missing %s server in %#v", typ, servers)
		}
		if srv["domain_resolver"] != ipS["tag"] {
			t.Errorf("%s domain_resolver = %v, want %v", typ, srv["domain_resolver"], ipS["tag"])
		}
		if _, ok := srv["tls"]; !ok {
			t.Errorf("%s server missing tls block (s-ui shape): %#v", typ, srv)
		}
	}
	// Regression guard: no server is addressed by a domain without a resolver.
	for _, s := range servers {
		m := s.(map[string]any)
		host, _ := m["server"].(string)
		if strings.TrimSpace(host) != "" && net.ParseIP(strings.TrimSpace(host)) == nil {
			if _, has := m["domain_resolver"]; !has {
				t.Errorf("domain server %q has no domain_resolver — sing-box would refuse to start: %#v", host, m)
			}
		}
	}
}

func TestMapXrayDNS_AllDomainAddsLocalBootstrap(t *testing.T) {
	// When every server is domain-addressed (no IP to reuse), a local bootstrap is
	// appended and referenced.
	raw := map[string]any{"servers": []any{"https://dns.google/dns-query"}}
	ruleSets := []any{}
	seen := map[string]struct{}{}
	out, _ := mapXrayDNS(raw, &ruleSets, seen)
	servers := out["servers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("servers = %d, want 2 (https + appended local bootstrap): %#v", len(servers), servers)
	}
	local := findServerByType(servers, "local")
	if local == nil {
		t.Fatalf("expected an appended local bootstrap, got %#v", servers)
	}
	https := findServerByType(servers, "https")
	if https == nil || https["domain_resolver"] != local["tag"] {
		t.Errorf("https domain_resolver = %v, want local tag %v", https["domain_resolver"], local["tag"])
	}
}

func TestMapXrayDNS_OutOfRangePortDropped(t *testing.T) {
	raw := map[string]any{"servers": []any{"udp://8.8.8.8:70000", "https://dns.google:99999/dns-query"}}
	ruleSets := []any{}
	seen := map[string]struct{}{}
	out, warnings := mapXrayDNS(raw, &ruleSets, seen)
	for _, s := range out["servers"].([]any) {
		if _, ok := s.(map[string]any)["server_port"]; ok {
			t.Errorf("out-of-range port must be omitted: %v", s)
		}
	}
	if !strings.Contains(strings.Join(warnings, "\n"), "out-of-range port") {
		t.Errorf("expected an out-of-range port warning, got %v", warnings)
	}
}

func TestMapXrayDNS_FinalFallbackWhenAllScoped(t *testing.T) {
	raw := map[string]any{"servers": []any{
		map[string]any{"address": "8.8.8.8", "domains": []any{"domain:a.com"}},
		map[string]any{"address": "1.1.1.1", "domains": []any{"domain:b.com"}},
	}}
	ruleSets := []any{}
	seen := map[string]struct{}{}
	out, _ := mapXrayDNS(raw, &ruleSets, seen)
	servers := out["servers"].([]any)
	if out["final"] == nil {
		t.Fatalf("final should fall back to the first server when all are domain-scoped")
	}
	if out["final"] != servers[0].(map[string]any)["tag"] {
		t.Errorf("final = %v, want first server tag %v", out["final"], servers[0].(map[string]any)["tag"])
	}
	if rules, _ := out["rules"].([]any); len(rules) != 2 {
		t.Errorf("expected 2 domain-scoped rules, got %d", len(rules))
	}
}

// TestMapXrayRouting_DNSIntegration verifies the DNS block is translated within
// MapXrayRouting and that a geosite used in a DNS rule registers its remote rule
// set at the shared route level.
func TestMapXrayRouting_DNSIntegration(t *testing.T) {
	raw := `{"dns":{"servers":["1.1.1.1",{"address":"8.8.8.8","domains":["geosite:cn"]}],"queryStrategy":"UseIPv6"}}`
	mapped, _, _, _ := MapXrayRouting(raw, nil)

	dns := mapped["dns"].(map[string]any)
	if dns["strategy"] != "ipv6_only" {
		t.Errorf("strategy = %v, want ipv6_only", dns["strategy"])
	}
	if servers, _ := dns["servers"].([]any); len(servers) != 2 {
		t.Fatalf("dns servers = %d, want 2", len(servers))
	}
	if rules, _ := dns["rules"].([]any); len(rules) != 1 {
		t.Fatalf("dns rules = %d, want 1", len(rules))
	}
	route := mapped["route"].(map[string]any)
	rsets := route["rule_set"].([]any)
	found := false
	for _, rs := range rsets {
		if rs.(map[string]any)["tag"] == "geosite-cn" {
			found = true
		}
	}
	if !found {
		t.Errorf("geosite-cn rule set not registered at route level: %v", rsets)
	}
}
