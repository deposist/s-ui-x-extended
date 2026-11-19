package service

import (
	"encoding/json"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// ClientNodeLinks returns the link entries that let a client reach every node it
// is assigned to.
//
// Links are built from the node's last announced inventory rather than stored on
// the client, so changing a node's address or inbounds does not require touching
// any client row. A disabled node, an unreachable one (no inventory yet), or one
// whose panel publishes no descriptors simply contributes nothing.
//
// The client's own inbound tags decide what may be published: the fleet projects
// a client's local inbounds onto matching node inbounds by tag, so only tags the
// client already has locally can appear.
func ClientNodeLinks(client *model.Client) []map[string]string {
	links, _ := ClientNodeLinksWithDeadline(client)
	return links
}

// ClientNodeLinksWithDeadline also returns the first instant a published peer
// link can expire without a database write.
func ClientNodeLinksWithDeadline(client *model.Client) ([]map[string]string, time.Time) {
	links := []map[string]string{}
	if client == nil {
		return links, time.Time{}
	}
	db := database.GetDB()
	if db == nil {
		return links, time.Time{}
	}
	paired, deadline := pairedClientLinks(db, client.Id)
	links = append(links, paired...)

	var clientInboundIDs []uint
	if err := json.Unmarshal(client.Inbounds, &clientInboundIDs); err != nil || len(clientInboundIDs) == 0 {
		return links, deadline
	}
	var inbounds []model.Inbound
	if err := db.Model(model.Inbound{}).Select("tag").Where("id in ?", clientInboundIDs).Find(&inbounds).Error; err != nil {
		return links, deadline
	}
	tags := make([]string, 0, len(inbounds))
	for _, inbound := range inbounds {
		tags = append(tags, inbound.Tag)
	}

	var assignments []model.ClientNode
	if err := db.Model(model.ClientNode{}).Where("client_id = ?", client.Id).Find(&assignments).Error; err != nil || len(assignments) == 0 {
		return links, deadline
	}
	uuids := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		uuids = append(uuids, assignment.NodeUUID)
	}
	var nodes []model.Node
	if err := db.Model(model.Node{}).Where("uuid in ?", uuids).Order("name").Find(&nodes).Error; err != nil {
		return links, deadline
	}
	for _, node := range nodes {
		if node.EnrollmentDigest != "" {
			continue
		}
		links = append(links, NodeLinks(client.Config, tags, node)...)
	}
	return links, deadline
}
