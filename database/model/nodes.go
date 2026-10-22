package model

import "encoding/json"

// Node is a managed server that receives clients from this panel. The row holds
// only what the controller owns; the fleet tables themselves live in the
// manager's own database.
type Node struct {
	Id uint `json:"id" gorm:"primaryKey;autoIncrement"`

	// UUID is the identity the node announces on connect. It is chosen when the
	// node record is created and never changes.
	UUID string `json:"uuid" gorm:"uniqueIndex;not null"`
	Name string `json:"name"`

	// LinkHost is the address clients dial for this node's inbounds. Empty means
	// the node's inbounds do not appear in subscriptions.
	LinkHost string `json:"linkHost"`

	Enabled bool `json:"enabled" gorm:"default:true;not null"`

	// Inventory is the last self-description the node announced, as sent by its
	// panel: one entry per inbound with the material needed to build a link.
	// Empty means the node has never connected.
	Inventory json.RawMessage `json:"inventory"`

	UpdatedAt int64 `json:"updatedAt"`
}

// ClientNode assigns a client to a node. A client without rows here is local
// only, which is exactly the pre-fleet behaviour.
type ClientNode struct {
	Id       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId uint   `json:"clientId" gorm:"uniqueIndex:idx_client_node;not null"`
	NodeUUID string `json:"nodeUuid" gorm:"uniqueIndex:idx_client_node;not null"`
}
