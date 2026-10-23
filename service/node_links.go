package service

import (
	"encoding/json"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util"
)

// NodeInboundDescriptor is the per-inbound payload a node's panel puts into its
// connect handshake. The core treats it as opaque; only panels read it, so both
// sides of a fleet agree on this shape and nothing else.
type NodeInboundDescriptor struct {
	// Hostname is the address clients dial for this node. The node's panel sets
	// it; the controller never guesses.
	Hostname string `json:"hostname"`

	// Inbound is the node's own inbound definition, in the same shape the local
	// link generator consumes, so a controller can build an identical link.
	Inbound json.RawMessage `json:"inbound"`
}

// NodeInventory is the decoded self-description of one node: its inbound tags
// and, per tag, the material needed to publish a link for it. Limiters are
// recorded alongside so the UI can offer only the servers that can enforce the
// per-user limits configured for a client.
type NodeInventory struct {
	Inbounds []NodeInventoryInbound `json:"inbounds"`
	Limiters []NodeInventoryLimiter `json:"limiters,omitempty"`
}

type NodeInventoryInbound struct {
	Tag        string `json:"tag"`
	Type       string `json:"type"`
	Descriptor string `json:"descriptor"`
}

type NodeInventoryLimiter struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Variant string `json:"variant"`
}

// DecodeNodeLimiters reads the limiter entries of an inventory blob.
func DecodeNodeLimiters(raw json.RawMessage) []NodeInventoryLimiter {
	return DecodeNodeInventory(raw).Limiters
}

// DecodeNodeInventory reads the inventory blob stored on a node row. A malformed
// or empty blob decodes to an empty inventory: a node that never connected, or
// one whose panel does not publish descriptors, simply contributes no links.
func DecodeNodeInventory(raw json.RawMessage) NodeInventory {
	var inventory NodeInventory
	if len(strings.TrimSpace(string(raw))) == 0 {
		return inventory
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		return NodeInventory{}
	}
	return inventory
}

// DecodeNodeInboundDescriptor reads one inbound descriptor.
func DecodeNodeInboundDescriptor(raw string) (NodeInboundDescriptor, bool) {
	var descriptor NodeInboundDescriptor
	if strings.TrimSpace(raw) == "" {
		return descriptor, false
	}
	if err := json.Unmarshal([]byte(raw), &descriptor); err != nil {
		return NodeInboundDescriptor{}, false
	}
	if len(strings.TrimSpace(string(descriptor.Inbound))) == 0 {
		return NodeInboundDescriptor{}, false
	}
	return descriptor, true
}

// NodeLinks builds the link entries a client needs to reach the given node.
//
// Only inbounds the client also reaches locally are published: the fleet
// projects a client's local inbounds onto matching node inbounds by tag, so a
// tag the client does not have locally is not a connection it may use.
func NodeLinks(clientConfig json.RawMessage, clientInboundTags []string, node model.Node) []map[string]string {
	links := []map[string]string{}
	if !node.Enabled || node.LinkHost == "" {
		return links
	}
	inventory := DecodeNodeInventory(node.Inventory)
	if len(inventory.Inbounds) == 0 {
		return links
	}
	wanted := make(map[string]struct{}, len(clientInboundTags))
	for _, tag := range clientInboundTags {
		wanted[tag] = struct{}{}
	}
	for _, entry := range inventory.Inbounds {
		if _, ok := wanted[entry.Tag]; !ok {
			continue
		}
		descriptor, ok := DecodeNodeInboundDescriptor(entry.Descriptor)
		if !ok {
			continue
		}
		var inbound model.Inbound
		if err := json.Unmarshal(descriptor.Inbound, &inbound); err != nil {
			continue
		}
		hostname := descriptor.Hostname
		if hostname == "" {
			hostname = node.LinkHost
		}
		for _, uri := range util.LinkGenerator(clientConfig, &inbound, hostname) {
			links = append(links, map[string]string{
				"remark": node.Name + " " + entry.Tag,
				"type":   "node",
				"uri":    uri,
			})
		}
	}
	return links
}
