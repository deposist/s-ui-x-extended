package service

import (
	"encoding/json"
	"math"
	"slices"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	peerTrafficSampleLimit = 256
	peerTrafficFreshness   = 5 * time.Minute
)

type PeerTrafficReport struct {
	SampledAt int64               `json:"sampledAt"`
	Samples   []PeerTrafficSample `json:"samples"`
}

type PeerTrafficSample struct {
	ClientID uint  `json:"clientId"`
	Used     int64 `json:"used"`
}

func localPeerTrafficReport(db *gorm.DB, panel model.NodePanel, now time.Time) *PeerTrafficReport {
	var users []peerUser
	if err := json.Unmarshal(panel.ManagedUsers, &users); err != nil {
		return nil
	}
	var ids []uint
	metered := make(map[uint]bool, len(users))
	for _, user := range users {
		if user.ProbeDestination == "" {
			ids = append(ids, user.ClientID)
			metered[user.ClientID] = user.Volume > 0
		}
	}
	if len(ids) == 0 {
		return nil
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	blocks := (len(ids) + peerTrafficSampleLimit - 1) / peerTrafficSampleLimit
	start := int((now.Unix()/15)%int64(blocks)) * peerTrafficSampleLimit
	ids = ids[start:min(start+peerTrafficSampleLimit, len(ids))]
	var counters []model.PeerTraffic
	if err := db.Where("node_uuid = ? AND client_id IN ?", panel.UUID, ids).Find(&counters).Error; err != nil {
		return nil
	}
	used := make(map[uint]int64, len(counters))
	for _, row := range counters {
		if !metered[row.ClientID] {
			used[row.ClientID] = row.UnmeteredUsed
		} else if row.ResetAt == 0 || row.ResetAt > now.Unix() {
			used[row.ClientID] = row.Used
		}
	}
	report := &PeerTrafficReport{SampledAt: now.Unix(), Samples: make([]PeerTrafficSample, 0, len(ids))}
	for _, id := range ids {
		report.Samples = append(report.Samples, PeerTrafficSample{ClientID: id, Used: used[id]})
	}
	return report
}

func acceptPeerTraffic(tx *gorm.DB, nodeUUID string, desired PeerSyncResponse, report *PeerTrafficReport, now time.Time) error {
	if report == nil || len(report.Samples) == 0 || len(report.Samples) > peerTrafficSampleLimit ||
		report.SampledAt < now.Add(-time.Minute).Unix() || report.SampledAt > now.Add(15*time.Second).Unix() {
		return nil
	}
	eligible := make(map[uint]bool, len(desired.Users))
	for _, user := range desired.Users {
		eligible[user.ClientID] = true
	}
	rows := make([]model.PeerTrafficSnapshot, 0, len(report.Samples))
	for _, sample := range report.Samples {
		if !eligible[sample.ClientID] || sample.Used < 0 {
			continue
		}
		delete(eligible, sample.ClientID)
		rows = append(rows, model.PeerTrafficSnapshot{NodeUUID: nodeUUID, ClientID: sample.ClientID, Used: sample.Used, SampledAt: report.SampledAt})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_uuid"}, {Name: "client_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"used", "sampled_at"}),
		Where:     clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "excluded.sampled_at > peer_traffic_snapshots.sampled_at"}}},
	}).CreateInBatches(rows, 100).Error
}

// Metered users debit the quota at route time. For unlimited users the stats
// transaction aggregates the core's existing counters without per-packet SQL.
func addUnmeteredPeerTraffic(tx *gorm.DB, deltas map[string]clientTrafficDelta) error {
	if len(deltas) == 0 {
		return nil
	}
	panel, err := readNodePanel(tx)
	if err != nil || panel.Role != "slave" || len(panel.ManagedUsers) == 0 {
		return err
	}
	var users []peerUser
	if err := json.Unmarshal(panel.ManagedUsers, &users); err != nil {
		return err
	}
	seen := make(map[uint]bool)
	for _, user := range users {
		if user.Volume != 0 || user.ProbeDestination != "" || seen[user.ClientID] {
			continue
		}
		delta := deltas[user.Name]
		if delta.up < 0 || delta.down < 0 || (delta.up == 0 && delta.down == 0) {
			continue
		}
		seen[user.ClientID] = true
		used := min(delta.up, math.MaxInt64-delta.down) + delta.down
		if err := tx.Exec(`INSERT INTO peer_traffics(node_uuid, client_id, used, unmetered_used) VALUES (?, ?, 0, ?)
			ON CONFLICT(node_uuid, client_id) DO UPDATE SET
				unmetered_used = peer_traffics.unmetered_used + MIN(excluded.unmetered_used, 9223372036854775807 - peer_traffics.unmetered_used)`,
			panel.UUID, user.ClientID, used).Error; err != nil {
			return err
		}
	}
	return nil
}
