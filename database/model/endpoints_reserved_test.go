package model

import (
	"encoding/json"
	"testing"
)

func TestEndpointMarshalDropsUnsupportedWireGuardPeerReserved(t *testing.T) {
	cases := []struct {
		name          string
		endpointType  string
		wantCoreType  string
		storedOptions json.RawMessage
	}{
		{
			name:         "warp",
			endpointType: "warp",
			wantCoreType: "wireguard",
			storedOptions: json.RawMessage(`{
				"address":["172.16.0.2/32"],
				"private_key":"private",
				"listen_port":0,
				"reserved":[1,2,3],
				"peers":[{
					"address":"162.159.192.1",
					"port":2408,
					"public_key":"peer",
					"allowed_ips":["0.0.0.0/0","::/0"],
					"reserved":[1,2,3]
				}]
			}`),
		},
		{
			name:         "wireguard",
			endpointType: "wireguard",
			wantCoreType: "wireguard",
			storedOptions: json.RawMessage(`{
				"address":["10.0.0.2/32"],
				"private_key":"private",
				"listen_port":0,
				"peers":[{
					"address":"198.51.100.1",
					"port":51820,
					"public_key":"peer",
					"allowed_ips":["0.0.0.0/0"],
					"reserved":[4,5,6]
				}]
			}`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := Endpoint{Type: tc.endpointType, Tag: tc.name + "-ep", Options: tc.storedOptions}
			raw, err := endpoint.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if got["type"] != tc.wantCoreType {
				t.Fatalf("type = %#v, want %q in %s", got["type"], tc.wantCoreType, string(raw))
			}
			if _, ok := got["reserved"]; ok {
				t.Fatalf("top-level reserved leaked into core endpoint JSON: %s", string(raw))
			}
			peers, ok := got["peers"].([]any)
			if !ok || len(peers) != 1 {
				t.Fatalf("peers = %#v, want one peer in %s", got["peers"], string(raw))
			}
			peer, ok := peers[0].(map[string]any)
			if !ok {
				t.Fatalf("peer = %#v", peers[0])
			}
			if _, ok := peer["reserved"]; ok {
				t.Fatalf("peer reserved leaked into core endpoint JSON: %s", string(raw))
			}
		})
	}
}

func TestWarpUnmarshalDropsReservedBeforeStore(t *testing.T) {
	payload := []byte(`{
		"type":"warp",
		"tag":"warp-ep",
		"address":["172.16.0.2/32"],
		"private_key":"private",
		"listen_port":0,
		"reserved":[7,8,9],
		"peers":[{
			"address":"162.159.192.1",
			"port":2408,
			"public_key":"peer",
			"allowed_ips":["0.0.0.0/0","::/0"],
			"reserved":[7,8,9]
		}]
	}`)
	var endpoint Endpoint
	if err := json.Unmarshal(payload, &endpoint); err != nil {
		t.Fatal(err)
	}
	var options map[string]any
	if err := json.Unmarshal(endpoint.Options, &options); err != nil {
		t.Fatal(err)
	}
	if _, ok := options["reserved"]; ok {
		t.Fatalf("warp top-level reserved should not be stored for wireguard-mode core JSON: %s", endpoint.Options)
	}
	peers := options["peers"].([]any)
	peer := peers[0].(map[string]any)
	if _, ok := peer["reserved"]; ok {
		t.Fatalf("warp peer reserved should not be stored for wireguard-mode core JSON: %s", endpoint.Options)
	}
}
