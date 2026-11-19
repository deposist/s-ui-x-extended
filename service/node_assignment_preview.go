package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrStalePeerPreview = errors.New("assignment changed; recalculate the preview")

type PeerAssignmentPreview struct {
	Changes     int                         `json:"changes"`
	Skipped     int                         `json:"skipped"`
	Fingerprint string                      `json:"fingerprint"`
	Items       []PeerAssignmentPreviewItem `json:"items"`
}

type PeerAssignmentPreviewItem struct {
	ClientID uint   `json:"clientId"`
	Name     string `json:"name,omitempty"`
	Effect   string `json:"effect"`
	Reason   string `json:"reason,omitempty"`
}

func previewPeerAssignments(db *gorm.DB, nodeUUID string, inboundID uint, ids []uint, assign bool, now time.Time) (PeerAssignmentPreview, []uint, error) {
	var result PeerAssignmentPreview
	if len(ids) == 0 || len(ids) > 500 {
		return result, nil, errors.New("select between 1 and 500 clients")
	}
	node, err := editablePeerInbound(db, nodeUUID, inboundID)
	if err != nil {
		return result, nil, err
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	inbounds := peerInboundsFromInventory(node.Inventory)
	var clients []struct {
		model.Client
		Limits model.ClientLimits `gorm:"embedded;embeddedPrefix:lim_"`
	}
	if err := db.Table("clients").
		Select(`clients.id, clients.name, clients.enable, clients.config, clients.expiry, clients.delay_start, clients.peer_delay_start,
			clients.auto_reset, clients.reset_days, clients.limit_ip, clients.ip_limit_mode, clients.volume,
			limits.speed_shared AS lim_speed_shared, limits.speed AS lim_speed, limits.download_speed AS lim_download_speed,
			limits.upload_speed AS lim_upload_speed, limits.connections AS lim_connections`).
		Joins("LEFT JOIN client_limits AS limits ON limits.client_id = clients.id").
		Where("clients.id IN ?", ids).Order("clients.id").Scan(&clients).Error; err != nil {
		return result, nil, err
	}
	names := make(map[uint]string, len(clients))
	for _, client := range clients {
		names[client.Id] = client.Name
	}
	var present []uint
	if err := db.Model(&model.PeerAssignment{}).
		Where("node_uuid = ? AND inbound_id = ? AND client_id IN ?", nodeUUID, inboundID, ids).
		Order("client_id").Pluck("client_id", &present).Error; err != nil {
		return result, nil, err
	}
	current, err := peerUsersForAssignments(db, peerAssignments(db, nodeUUID), inbounds, now, !node.PeerLeaseCapable)
	if err != nil {
		return result, nil, err
	}
	eligible := make(map[uint]bool)
	if assign {
		proposed, err := peerUsersForAssignments(db, assignmentsWithChanges(db, nodeUUID, inboundID, ids, true), inbounds, now, !node.PeerLeaseCapable)
		if err != nil {
			return result, nil, err
		}
		for _, user := range proposed.Users {
			if user.InboundID == inboundID {
				eligible[user.ClientID] = true
			}
		}
	}
	var changes []uint
	for _, id := range ids {
		_, assigned := slices.BinarySearch(present, id)
		item := PeerAssignmentPreviewItem{ClientID: id, Name: names[id], Effect: "none"}
		switch {
		case assign && assigned:
			item.Reason = "already_assigned"
		case !assign && !assigned:
			item.Reason = "not_assigned"
		case assign && !eligible[id]:
			item.Reason = "incompatible_policy"
			if _, found := names[id]; !found {
				item.Reason = "missing_client"
			}
		default:
			if assign {
				item.Effect = "link_added"
				if node.LinkHost == "" {
					item.Effect, item.Reason = "none", "missing_address"
				}
			} else {
				item.Effect = "link_removed"
			}
			changes = append(changes, id)
		}
		result.Items = append(result.Items, item)
	}
	result.Changes, result.Skipped = len(changes), len(ids)-len(changes)
	next, err := peerUsersForAssignments(db, assignmentsWithChanges(db, nodeUUID, inboundID, changes, assign), inbounds, now, !node.PeerLeaseCapable)
	if err != nil {
		return result, nil, err
	}
	after := slices.Clone(present)
	if assign {
		after = append(after, changes...)
	} else {
		after = slices.DeleteFunc(after, func(id uint) bool { _, remove := slices.BinarySearch(changes, id); return remove })
	}
	slices.Sort(after)
	context, err := json.Marshal([]any{nodeUUID, inboundID, ids, assign, node.LinkHost, node.Inventory, clients})
	if err != nil {
		return result, nil, err
	}
	fingerprint := func(revision string, membership []uint) string {
		encoded, _ := json.Marshal(append([]uint{}, membership...))
		mac := hmac.New(sha256.New, []byte(node.EnrollmentDigest))
		mac.Write([]byte("peer-assignment-preview-v1\x00"))
		mac.Write(context)
		mac.Write([]byte(revision))
		mac.Write(encoded)
		return hex.EncodeToString(mac.Sum(nil))
	}
	result.Fingerprint = fingerprint(current.Revision, present) + "." + fingerprint(next.Revision, after)
	return result, changes, nil
}

func assignmentsWithChanges(db *gorm.DB, nodeUUID string, inboundID uint, ids []uint, assign bool) *gorm.DB {
	current := peerAssignments(db, nodeUUID)
	if len(ids) == 0 {
		return current
	}
	if !assign {
		return current.Where("NOT (inbound_id = ? AND client_id IN ?)", inboundID, ids)
	}
	return db.Raw("SELECT inbound_id, client_id FROM peer_assignments WHERE node_uuid = ? UNION SELECT ?, id FROM clients WHERE id IN ?", nodeUUID, inboundID, ids)
}

func (s *NodeService) PreviewPeerClients(uuid string, inboundID uint, ids []uint, assign bool) (PeerAssignmentPreview, error) {
	db := s.db()
	if db == nil {
		return PeerAssignmentPreview{}, errors.New("database is not initialized")
	}
	var preview PeerAssignmentPreview
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		preview, _, err = previewPeerAssignments(tx, uuid, inboundID, ids, assign, s.peerTime())
		return err
	})
	return preview, err
}

func (s *NodeService) ApplyPeerClientPreview(uuid string, inboundID uint, ids []uint, assign bool, fingerprint string) (int, error) {
	db := s.db()
	if db == nil {
		return 0, errors.New("database is not initialized")
	}
	changed := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		preview, ids, err := previewPeerAssignments(tx, uuid, inboundID, ids, assign, s.peerTime())
		if err != nil {
			if errors.Is(err, errPeerInboundUnavailable) {
				return ErrStalePeerPreview
			}
			return err
		}
		if preview.Fingerprint != fingerprint {
			before, _, _ := strings.Cut(preview.Fingerprint, ".")
			_, after, ok := strings.Cut(fingerprint, ".")
			if ok && before == after && len(ids) == 0 {
				return nil
			}
			return ErrStalePeerPreview
		}
		if len(ids) == 0 {
			return nil
		}
		if !assign {
			result := tx.Where("node_uuid = ? AND inbound_id = ? AND client_id IN ?", uuid, inboundID, ids).Delete(model.PeerAssignment{})
			changed = int(result.RowsAffected)
			return result.Error
		}
		rows := make([]model.PeerAssignment, 0, len(ids))
		for _, id := range ids {
			rows = append(rows, model.PeerAssignment{NodeUUID: uuid, InboundID: inboundID, ClientID: id})
		}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(rows, 100)
		changed = int(result.RowsAffected)
		return result.Error
	})
	if err != nil {
		return 0, err
	}
	if changed > 0 && invalidateSubscriptionCacheAfterSave != nil {
		invalidateSubscriptionCacheAfterSave()
	}
	return changed, nil
}
