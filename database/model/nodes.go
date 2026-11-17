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

	// LimiterRoles maps operator-selected outbounds to the per-client limiter roles.
	// The column name stays stable for existing databases.
	LimiterRoles json.RawMessage `json:"limiterRoles" gorm:"column:limiters"`

	UpdatedAt int64 `json:"updatedAt"`
}

// NodeLimiterTags maps a limiter role to the outbound tag that enforces it on
// one node. Roles are independent: a node may enforce speed but not traffic.
type NodeLimiterTags struct {
	// SpeedShared is a single bandwidth limiter capped in both directions.
	SpeedShared string `json:"speedShared"`
	// SpeedDown and SpeedUp carry one direction each when the operator splits
	// the limit. The core stores one number per limiter record, so a split
	// limit needs two outbounds.
	SpeedDown string `json:"speedDown"`
	SpeedUp   string `json:"speedUp"`

	// Connections caps concurrent connections per user.
	Connections string `json:"connections"`
}

// FleetNodeOwnership records the manager squad created for a node by this panel.
type FleetNodeOwnership struct {
	NodeUUID string `json:"nodeUuid" gorm:"primaryKey"`
	SquadID  int    `json:"squadId" gorm:"not null"`
}

// ClientNode assigns a client to a node. A client without rows here is local
// only, which is exactly the pre-fleet behaviour.
type ClientNode struct {
	Id       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId uint   `json:"clientId" gorm:"uniqueIndex:idx_client_node;not null"`
	NodeUUID string `json:"nodeUuid" gorm:"uniqueIndex:idx_client_node;not null"`
}
