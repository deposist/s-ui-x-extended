package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func nodeInboundDescriptorJSON(t *testing.T, hostname string, inboundJSON string) string {
	t.Helper()
	descriptor := map[string]interface{}{
		"hostname": hostname,
		"inbound":  json.RawMessage(inboundJSON),
	}
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	return string(encoded)
}

func nodeWithInventory(t *testing.T, name, linkHost string, enabled bool, inbounds []NodeInventoryInbound) model.Node {
	t.Helper()
	inventory, err := json.Marshal(NodeInventory{Inbounds: inbounds})
	if err != nil {
		t.Fatalf("marshal inventory: %v", err)
	}
	return model.Node{Name: name, LinkHost: linkHost, Enabled: enabled, Inventory: inventory}
}

const vlessInboundForNode = `{"id":7,"type":"vless","tag":"vless-443","listen":"::","listen_port":443,"tls_id":0,"addrs":[],"out_json":{}}`

// TestNodeLinksPublishAssignedInbounds is the subscription requirement: a client
// must reach the main server and every slave, so a node contributes links for
// the inbounds the client also has locally.
func TestNodeLinksPublishAssignedInbounds(t *testing.T) {
	descriptor := nodeInboundDescriptorJSON(t, "de1.example.com", vlessInboundForNode)
	node := nodeWithInventory(t, "Германия-1", "de1.example.com", true, []NodeInventoryInbound{{
		Tag:        "vless-443",
		Type:       "vless",
		Descriptor: descriptor,
	}})

	config := json.RawMessage(`{"vless":{"uuid":"11111111-2222-3333-4444-555555555555","flow":""}}`)
	links := NodeLinks(config, []string{"vless-443"}, node)
	if len(links) != 1 {
		t.Fatalf("expected one link, got %d: %+v", len(links), links)
	}
	if links[0]["type"] != "node" {
		t.Fatalf("link type = %q, want node", links[0]["type"])
	}
	if links[0]["remark"] != "Германия-1 vless-443" {
		t.Fatalf("remark = %q", links[0]["remark"])
	}
	if got := links[0]["uri"]; len(got) < 8 || got[:8] != "vless://" {
		t.Fatalf("uri = %q, want a vless link", got)
	}
	if !contains(got0(links), "de1.example.com") {
		t.Fatalf("link should point at the node address: %q", links[0]["uri"])
	}
}

func got0(links []map[string]string) string {
	if len(links) == 0 {
		return ""
	}
	return links[0]["uri"]
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// TestNodeLinksSkipUnassignedInbounds keeps the projection honest: a node
// inbound the client does not have locally is not a connection it may use.
func TestNodeLinksSkipUnassignedInbounds(t *testing.T) {
	node := nodeWithInventory(t, "NL", "nl.example.com", true, []NodeInventoryInbound{{
		Tag:        "vless-443",
		Type:       "vless",
		Descriptor: nodeInboundDescriptorJSON(t, "nl.example.com", vlessInboundForNode),
	}})
	if links := NodeLinks(json.RawMessage(`{}`), []string{"trojan-2087"}, node); len(links) != 0 {
		t.Fatalf("expected no links for an unassigned inbound, got %+v", links)
	}
}

// TestNodeLinksSkipDisabledAndEmpty covers the states a controller sees before a
// node ever connects.
func TestNodeLinksSkipDisabledAndEmpty(t *testing.T) {
	descriptor := nodeInboundDescriptorJSON(t, "de1.example.com", vlessInboundForNode)
	disabled := nodeWithInventory(t, "DE", "de1.example.com", false, []NodeInventoryInbound{{
		Tag: "vless-443", Type: "vless", Descriptor: descriptor,
	}})
	if links := NodeLinks(json.RawMessage(`{}`), []string{"vless-443"}, disabled); len(links) != 0 {
		t.Fatalf("disabled node must publish nothing, got %+v", links)
	}

	never := model.Node{Name: "FI", LinkHost: "fi.example.com", Enabled: true}
	if links := NodeLinks(json.RawMessage(`{}`), []string{"vless-443"}, never); len(links) != 0 {
		t.Fatalf("node without inventory must publish nothing, got %+v", links)
	}

	noHost := nodeWithInventory(t, "SE", "", true, []NodeInventoryInbound{{
		Tag: "vless-443", Type: "vless", Descriptor: descriptor,
	}})
	if links := NodeLinks(json.RawMessage(`{}`), []string{"vless-443"}, noHost); len(links) != 0 {
		t.Fatalf("node without a link host must publish nothing, got %+v", links)
	}
}
