package service

import (
	"crypto/sha256"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

type peerProjectionKey struct {
	db   *gorm.DB
	uuid string
}

type peerProjection struct {
	input     [sha256.Size]byte
	inventory string
	legacy    bool
	state     PeerSyncResponse
}

var peerProjections = struct {
	sync.Mutex
	entries map[peerProjectionKey]peerProjection
}{entries: make(map[peerProjectionKey]peerProjection)}

func init() {
	database.RegisterResetHook("service.peer_projections", func() error {
		peerProjections.Lock()
		clear(peerProjections.entries)
		peerProjections.Unlock()
		return nil
	})
}

func projectedPeerState(db *gorm.DB, node *model.Node, inbounds []PeerInbound, now time.Time) (PeerSyncResponse, error) {
	key := peerProjectionKey{db, node.UUID}
	projection := peerProjection{inventory: peerInventoryDigest(node.Inventory, node.LinkHost), legacy: !node.PeerLeaseCapable}
	err := db.Transaction(func(tx *gorm.DB) error {
		assignments := peerAssignments(tx, node.UUID)
		// Compare the inputs read from SQLite on this request, including expiry
		// eligibility. The digest covers policy edits and deletions immediately.
		var input []byte
		if err := tx.Raw(`SELECT json_group_array(json_array(inbound_id, id, name, hex(config), expiry,
			delay_start, peer_delay_start, auto_reset, reset_days, limit_ip, ip_limit_mode, volume,
			lim_speed_shared, lim_speed, lim_download_speed, lim_upload_speed, lim_connections,
			expiry > 0 AND expiry <= ?))
			FROM (?)`, now.Unix(), peerAssignmentRows(tx, assignments)).Row().Scan(&input); err != nil {
			return err
		}
		projection.input = sha256.Sum256(input)
		peerProjections.Lock()
		previous, exists := peerProjections.entries[key]
		peerProjections.Unlock()
		if exists && previous.input == projection.input && previous.inventory == projection.inventory && previous.legacy == projection.legacy {
			projection.state = previous.state
			return nil
		}
		state, err := buildPeerUsersForAssignments(tx, assignments, inbounds, now, projection.legacy, projection.inventory)
		projection.state = state
		return err
	})
	if err == nil {
		peerProjections.Lock()
		if _, exists := peerProjections.entries[key]; !exists && len(peerProjections.entries) >= 16 {
			for entry := range peerProjections.entries {
				delete(peerProjections.entries, entry)
				break
			}
		}
		peerProjections.entries[key] = projection
		peerProjections.Unlock()
	}
	return projection.state, err
}
