package service

import (
	"net/url"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

const PeerApplyHeader = "X-Node-Apply-Failed"

func nodeIncident(node model.Node, probe PeerProbeResult, now time.Time) string {
	if !node.Enabled || node.PeerDisconnect || node.LastPeerSync == 0 {
		return ""
	}
	if now.Unix()-node.LastPeerSync >= 45 {
		return "offline"
	}
	if node.PeerApplyFailed || node.PeerBlocked > 0 || (node.PeerLeaseCapable && (node.PeerAccessUntil != 0 && node.PeerAccessUntil <= now.Unix() ||
		node.PeerPendingSince != 0 && now.Unix()-node.PeerPendingSince >= 45)) {
		return "apply_failed"
	}
	if probe.Result != "" && probe.Result != "success" && probe.Result != "running" {
		return "probe_failed"
	}
	return ""
}

func (s *NodeService) ReconcileNodeIncidents() error {
	db := s.db()
	panel, err := readNodePanel(db)
	if err != nil || panel.Role != "master" {
		return err
	}
	now := s.peerTime()
	type transition struct {
		node  model.Node
		state string
	}
	var changes []transition
	err = db.Transaction(func(tx *gorm.DB) error {
		var nodes []model.Node
		if err := tx.Where("enrollment_digest <> ''").Find(&nodes).Error; err != nil {
			return err
		}
		for _, node := range nodes {
			probe := lastNodeProbe(collectNodeProbeResults(tx, &s.runtime().nodeProbes, &node), 0)
			state := nodeIncident(node, probe, now)
			if state == node.PeerIncident {
				continue
			}
			if err := tx.Model(&model.Node{}).Where("id = ?", node.Id).
				UpdateColumns(map[string]any{"peer_incident": state, "peer_incident_at": now.Unix()}).Error; err != nil {
				return err
			}
			if !node.Enabled || node.PeerDisconnect {
				continue
			}
			event := state
			if event == "" {
				event = "recovered"
			}
			audit, err := buildAuditRecord(AuditEvent{Actor: "system", Event: "node_incident_" + event, Resource: "node",
				Severity: AuditSeverityWarn, Details: map[string]any{"uuid": node.UUID, "name": node.Name, "previous": node.PeerIncident}})
			if err != nil {
				return err
			}
			audit.DateTime = now.Unix()
			if err := tx.Create(&audit).Error; err != nil {
				return err
			}
			changes = append(changes, transition{node, event})
		}
		return nil
	})
	if err != nil || len(changes) == 0 {
		return err
	}
	settings := &SettingService{}
	enabled, _ := settings.getBool("telegramNotifyNodes")
	if !enabled {
		return nil
	}
	panelURL, _ := settings.GetWebURI()
	telegram := &TelegramService{Runtime: s.runtime()}
	for _, change := range changes {
		telegram.NotifyTelegramEvent("node_"+change.state, map[string]string{
			"node": change.node.Name, "time": now.Format(time.RFC3339), "panel": nodeDetailsURL(panelURL, change.node.UUID),
		})
	}
	return nil
}

func nodeDetailsURL(base, nodeUUID string) string {
	address, err := url.Parse(base)
	if err != nil || (address.Scheme != "https" && address.Scheme != "http") || address.Hostname() == "" ||
		address.User != nil || address.RawQuery != "" || address.Fragment != "" {
		return ""
	}
	address = address.JoinPath("nodes")
	address.RawQuery = "node=" + url.QueryEscape(nodeUUID)
	return address.String()
}

func lastNodeProbe(results []PeerProbeResult, inboundID uint) PeerProbeResult {
	var latest PeerProbeResult
	for _, probe := range results {
		if (inboundID == 0 || probe.InboundID == inboundID) && probe.Result != "running" &&
			(probe.CheckedAt > latest.CheckedAt || probe.CheckedAt == latest.CheckedAt && probe.InboundID < latest.InboundID) {
			latest = probe
		}
	}
	return latest
}
