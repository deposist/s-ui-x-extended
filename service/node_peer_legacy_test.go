package service

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNodeLegacyInventoryKeepsInboundIDsAndSuppressesNewDescriptors(t *testing.T) {
	input := []PeerInbound{
		{ID: 1, Tag: "tcp", Type: "vless", Link: &peerPublicLink{Port: 443, Transport: &peerPublicTransport{Type: "tcp"}}},
		{ID: 2, Tag: "edge", Type: "vless", Link: &peerPublicLink{Port: 8443, Transport: &peerPublicTransport{Type: "ws"}}},
		{ID: 3, Tag: "fast", Type: "tuic", Link: &peerPublicLink{Port: 8444}},
	}
	body, err := json.Marshal(legacyPeerInventory(input))
	if err != nil {
		t.Fatal(err)
	}
	var beta9 []struct {
		ID   uint
		Tag  string
		Type string
		Link *struct{ Port uint16 }
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&beta9); err != nil {
		t.Fatal(err)
	}
	if len(beta9) != 3 || beta9[0].Link == nil || beta9[1].ID != 2 || beta9[1].Link != nil || beta9[2].Link != nil {
		t.Fatalf("unexpected legacy inventory: %+v", beta9)
	}
}
