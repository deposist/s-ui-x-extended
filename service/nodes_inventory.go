package service

import (
	"bytes"
	"encoding/json"

	"github.com/deposist/s-ui-x-extended/database/model"

	CM "github.com/sagernet/sing-box/service/manager/constant"
	"gorm.io/gorm"
)

// SyncInventory stores the inventory every connected node announced, so the
// controller knows what each node accepts without asking it again.
//
// The manager keeps hellos in memory only, so this runs when the node list is
// read: after a core restart the rows keep the last known inventory until the
// node reconnects, which is exactly what the UI should show.
func (s *NodeService) SyncInventory() (int, error) {
	db := s.db()
	if db == nil {
		return 0, nil
	}
	coreInstance := s.runtime().Core()
	if coreInstance == nil {
		return 0, nil
	}
	var nodes []model.Node
	if err := db.Model(model.Node{}).Where("COALESCE(enrollment_digest, '') = ''").Select("uuid", "inventory").Find(&nodes).Error; err != nil {
		return 0, err
	}
	if len(nodes) == 0 {
		return 0, nil
	}
	// The list endpoint syncs on every load, so only rows that actually changed
	// are written: a settled fleet must not rewrite every node row each time.
	changed := make(map[string]json.RawMessage, len(nodes))
	err := coreInstance.WithFleetManager(func(manager CM.Manager) error {
		reader, ok := manager.(interface {
			GetNodeHello(string) (CM.NodeHello, bool)
		})
		if !ok {
			// An older manager has no hellos to report; the stored rows keep
			// whatever they already have rather than being cleared.
			return nil
		}
		for _, node := range nodes {
			hello, found := reader.GetNodeHello(node.UUID)
			if !found {
				continue
			}
			inventory, err := encodeNodeInventory(hello)
			if err != nil {
				return err
			}
			if bytes.Equal(bytes.TrimSpace(node.Inventory), bytes.TrimSpace(inventory)) {
				continue
			}
			changed[node.UUID] = inventory
		}
		return nil
	})
	if err != nil || len(changed) == 0 {
		return 0, err
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for uuid, inventory := range changed {
			if err := tx.Model(model.Node{}).Where("uuid = ?", uuid).Update("inventory", inventory).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return len(changed), nil
}

// encodeNodeInventory turns a node announcement into the stored blob.
func encodeNodeInventory(hello CM.NodeHello) (json.RawMessage, error) {
	inventory := NodeInventory{}
	for _, inbound := range hello.Inbounds {
		inventory.Inbounds = append(inventory.Inbounds, NodeInventoryInbound{
			Tag:        inbound.Tag,
			Type:       inbound.Type,
			Descriptor: inbound.Descriptor,
		})
	}
	for _, limiter := range hello.Limiters {
		inventory.Limiters = append(inventory.Limiters, NodeInventoryLimiter{
			Tag:     limiter.Tag,
			Type:    limiter.Type,
			Variant: limiter.Variant,
		})
	}
	return json.Marshal(inventory)
}
