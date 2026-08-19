package model

import (
	"encoding/json"
	"testing"
)

func TestOutboundJSONDropsLegacyCrossLeakFields(t *testing.T) {
	outbound := Outbound{
		Type: "direct",
		Tag:  "direct-out",
		Options: json.RawMessage(`{
			"server":"example.com",
			"server_port":443,
			"domain_strategy":"prefer_ipv4",
			"proxy_protocol":true,
			"proxy_protocol_accept_no_header":true,
			"udp_disable_domain_unmapping":true
		}`),
	}

	data, err := json.Marshal(outbound)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"proxy_protocol",
		"proxy_protocol_accept_no_header",
		"udp_disable_domain_unmapping",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("legacy outbound key %q leaked into core JSON: %s", key, data)
		}
	}
	for _, key := range []string{"server", "server_port", "domain_strategy"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("valid outbound key %q was dropped: %s", key, data)
		}
	}
}
