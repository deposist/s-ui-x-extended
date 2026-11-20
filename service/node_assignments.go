package service

import (
	"errors"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errPeerInboundUnavailable = errors.New("the selected Slave inbound no longer exists")

func checkPeerInbound(db *gorm.DB, uuid string, inboundID uint) error {
	_, err := editablePeerInbound(db, uuid, inboundID)
	return err
}

func editablePeerInbound(db *gorm.DB, uuid string, inboundID uint) (*model.Node, error) {
	panel, err := readNodePanel(db)
	if err != nil {
		return nil, err
	}
	if panel.Role != "master" || inboundID == 0 {
		return nil, errors.New("select a Slave inbound on Master")
	}
	var node model.Node
	if err := db.Where("uuid = ? AND enabled AND enrollment_digest <> '' AND NOT COALESCE(peer_disconnect, 0)", uuid).Take(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errPeerInboundUnavailable
		}
		return nil, err
	}
	for _, inbound := range DecodeNodeInventory(node.Inventory).Inbounds {
		if inbound.ID == inboundID {
			return &node, nil
		}
	}
	return nil, errPeerInboundUnavailable
}

func (s *NodeService) PeerClients(uuid string, inboundID uint, limit, offset int, query string, onlyAssigned bool) (*ClientPage, error) {
	db := s.db()
	node, err := editablePeerInbound(db, uuid, inboundID)
	if err != nil {
		return nil, err
	}
	page, err := nodeClients(db, db.Model(model.PeerAssignment{}).Where("node_uuid = ? AND inbound_id = ?", uuid, inboundID), limit, offset, query, onlyAssigned)
	if err != nil || len(page.Items) == 0 {
		return page, err
	}
	inbounds := peerInboundsFromInventory(node.Inventory)
	now := s.peerTime()
	desired, err := projectedPeerState(db, node, inbounds, now)
	if err != nil {
		return nil, err
	}
	var inbound PeerInbound
	for _, candidate := range inbounds {
		if candidate.ID == inboundID {
			inbound = candidate
			break
		}
	}
	eligible := make(map[uint]bool)
	for _, user := range desired.Users {
		if user.InboundID == inboundID {
			eligible[user.ClientID] = true
		}
	}
	for i, item := range page.Items {
		if item.Assigned {
			page.Items[i].LinkReason = peerAssignmentLinkReason(node, inbound, eligible[item.Id], desired.Revision == node.PeerRevision, now)
		}
	}
	return page, nil
}

func (s *NodeService) ChangePeerClients(uuid string, inboundID uint, ids []uint, assign bool) (int, error) {
	db := s.db()
	if db == nil || len(ids) == 0 || len(ids) > 500 {
		return 0, errors.New("select between 1 and 500 clients")
	}
	changed := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := checkPeerInbound(tx, uuid, inboundID); err != nil {
			return err
		}
		if !assign {
			result := tx.Where("node_uuid = ? AND inbound_id = ? AND client_id IN ?", uuid, inboundID, ids).Delete(model.PeerAssignment{})
			changed = int(result.RowsAffected)
			return result.Error
		}
		var existing []uint
		if err := tx.Model(model.Client{}).Where("id IN ?", ids).Pluck("id", &existing).Error; err != nil {
			return err
		}
		wanted := make(map[uint]struct{}, len(ids))
		for _, id := range ids {
			wanted[id] = struct{}{}
		}
		if len(existing) != len(wanted) {
			return errors.New("one or more selected clients no longer exist")
		}
		rows := make([]model.PeerAssignment, 0, len(existing))
		for _, id := range existing {
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
