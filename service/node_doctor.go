package service

import (
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

type DoctorNodeAssignment struct {
	NodeUUID   string           `json:"nodeUuid"`
	NodeName   string           `json:"nodeName"`
	InboundID  uint             `json:"inboundId,omitempty"`
	InboundTag string           `json:"inboundTag,omitempty"`
	LinkReason string           `json:"linkReason"`
	Probe      *PeerProbeResult `json:"probe,omitempty"`
	QuotaState string           `json:"quotaState"`
	Used       *int64           `json:"used,omitempty"`
	Remaining  *int64           `json:"remaining,omitempty"`
	SampledAt  int64            `json:"sampledAt,omitempty"`
}

func (s *DoctorService) clientNodeDiagnostics(db *gorm.DB, client model.Client, now time.Time) ([]DoctorNodeAssignment, error) {
	var assignments []model.PeerAssignment
	if err := db.Where("client_id = ?", client.Id).Order("inbound_id").Find(&assignments).Error; err != nil {
		return nil, err
	}
	byNode := make(map[string][]uint, len(assignments))
	for _, assignment := range assignments {
		byNode[assignment.NodeUUID] = append(byNode[assignment.NodeUUID], assignment.InboundID)
	}
	var nodes []model.Node
	paired := db.Model(&model.PeerAssignment{}).Select("node_uuid").Where("client_id = ?", client.Id)
	manual := db.Model(&model.ClientNode{}).Select("node_uuid").Where("client_id = ?", client.Id)
	if err := db.Where("uuid IN (?) OR (COALESCE(enrollment_digest, '') = '' AND uuid IN (?))", paired, manual).Order("name, id").Find(&nodes).Error; err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	var samples []model.PeerTrafficSnapshot
	if err := db.Where("client_id = ?", client.Id).Find(&samples).Error; err != nil {
		return nil, err
	}
	bySample := make(map[string]model.PeerTrafficSnapshot, len(samples))
	for _, sample := range samples {
		bySample[sample.NodeUUID] = sample
	}
	var result []DoctorNodeAssignment
	for _, node := range nodes {
		if node.EnrollmentDigest == "" {
			reason := "unknown"
			if !node.Enabled {
				reason = "node_disabled"
			}
			result = append(result, DoctorNodeAssignment{NodeUUID: node.UUID, NodeName: node.Name, LinkReason: reason, QuotaState: "unknown"})
			continue
		}
		inbounds := peerInboundsFromInventory(node.Inventory)
		desired, err := peerUsersForAssignments(db, peerAssignments(db, node.UUID), inbounds, now, !node.PeerLeaseCapable)
		if err != nil {
			return nil, err
		}
		if err := desired.bindInventory(node.Inventory, node.LinkHost); err != nil {
			return nil, err
		}
		byInbound := make(map[uint]PeerInbound, len(inbounds))
		for _, inbound := range inbounds {
			byInbound[inbound.ID] = inbound
		}
		eligible := make(map[uint]bool)
		for _, user := range desired.Users {
			if user.ClientID == client.Id {
				eligible[user.InboundID] = true
			}
		}
		probes := collectNodeProbeResults(db, &s.runtime().nodeProbes, &node)
		for _, id := range byNode[node.UUID] {
			inbound := byInbound[id]
			item := DoctorNodeAssignment{NodeUUID: node.UUID, NodeName: node.Name, InboundID: id, InboundTag: inbound.Tag,
				LinkReason: peerAssignmentLinkReason(&node, inbound, eligible[id], desired.Revision == node.PeerRevision, now), QuotaState: "unknown"}
			if last := lastNodeProbe(probes, id); last.Result != "" {
				item.Probe = &last
			}
			if sample := bySample[node.UUID]; sample.SampledAt != 0 {
				item.SampledAt = sample.SampledAt
				item.QuotaState = "stale"
				if sample.SampledAt > now.Add(-peerTrafficFreshness).Unix() {
					item.Used, item.QuotaState = &sample.Used, "unlimited"
					if client.Volume > 0 {
						remaining := max(int64(0), client.Volume-sample.Used)
						item.Remaining, item.QuotaState = &remaining, "measured"
						if remaining == 0 {
							item.QuotaState = "exhausted"
						}
					}
				}
			}
			result = append(result, item)
		}
	}
	return result, nil
}
