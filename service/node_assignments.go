package service

import (
	"errors"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func checkPeerInbound(db *gorm.DB, uuid string, inboundID uint) error {
	panel, err := readNodePanel(db)
	if err != nil {
		return err
	}
	if panel.Role != "master" || inboundID == 0 {
		return errors.New("select a Slave inbound on Master")
	}
	var node model.Node
	if err := db.Where("uuid = ? AND enabled AND enrollment_digest <> ''", uuid).Take(&node).Error; err != nil {
		return err
	}
	for _, inbound := range DecodeNodeInventory(node.Inventory).Inbounds {
		if inbound.ID == inboundID {
			return nil
		}
	}
	return errors.New("the selected Slave inbound no longer exists")
}

func (s *NodeService) PeerClients(uuid string, inboundID uint, limit, offset int, query string, onlyAssigned bool) (*ClientPage, error) {
	db := s.db()
	if err := checkPeerInbound(db, uuid, inboundID); err != nil {
		return nil, err
	}
	return nodeClients(db, db.Model(model.PeerAssignment{}).Where("node_uuid = ? AND inbound_id = ?", uuid, inboundID), limit, offset, query, onlyAssigned)
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
