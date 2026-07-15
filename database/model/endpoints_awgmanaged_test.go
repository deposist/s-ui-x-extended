package model

import (
	"encoding/json"
	"testing"
)

// The frontend echoes the panel-only awgManaged marker (added by
// EndpointService.GetAll) back on save. It must never be stored in Options or
// reach the sing-box config: sing-box rejects unknown fields and fails to
// start ("endpoints[N].awgManaged: json: unknown field").
func TestEndpointUnmarshalDropsAWGManagedMarker(t *testing.T) {
	payload := []byte(`{
		"type":"wireguard",
		"tag":"wg-managed",
		"awgManaged":true,
		"address":["10.0.0.1/24"],
		"private_key":"private",
		"listen_port":51820,
		"peers":[],
		"ext":{"managed":true,"publicEndpoint":"vpn.example.com:51820","dns":["1.1.1.1"],"defaultDeviceLimit":3}
	}`)
	var endpoint Endpoint
	if err := json.Unmarshal(payload, &endpoint); err != nil {
		t.Fatal(err)
	}
	var options map[string]any
	if err := json.Unmarshal(endpoint.Options, &options); err != nil {
		t.Fatal(err)
	}
	if _, ok := options["awgManaged"]; ok {
		t.Fatalf("awgManaged leaked into stored Options: %s", endpoint.Options)
	}
}

// Rows poisoned before UnmarshalJSON stripped the marker must be healed when
// the core config is rendered.
func TestEndpointMarshalDropsAWGManagedMarker(t *testing.T) {
	stored := json.RawMessage(`{
		"address":["10.0.0.1/24"],
		"private_key":"private",
		"listen_port":51820,
		"awgManaged":true,
		"peers":[]
	}`)
	endpoint := Endpoint{Type: "wireguard", Tag: "wg-managed", Options: stored}
	raw, err := endpoint.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["awgManaged"]; ok {
		t.Fatalf("awgManaged leaked into core endpoint JSON: %s", string(raw))
	}
}
