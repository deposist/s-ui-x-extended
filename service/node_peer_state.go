package service

import (
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/ssrf"
	"gorm.io/gorm"
)

const maxPeerUsers = 4096

type peerUser struct {
	InboundID        uint   `json:"inboundId"`
	ClientID         uint   `json:"clientId"`
	Type             string `json:"type"`
	Name             string `json:"name"`
	UUID             string `json:"uuid,omitempty"`
	Password         string `json:"password,omitempty"`
	Flow             string `json:"flow,omitempty"`
	Expiry           int64  `json:"expiry,omitempty"`
	Connections      uint32 `json:"connections,omitempty"`
	Speed            uint64 `json:"speed,omitempty"`
	Download         uint64 `json:"downloadSpeed,omitempty"`
	Upload           uint64 `json:"uploadSpeed,omitempty"`
	Volume           int64  `json:"volume,omitempty"`
	LimitIP          int    `json:"limitIp,omitempty"`
	IPLimitMode      string `json:"ipLimitMode,omitempty"`
	DelayStart       bool   `json:"delayStart,omitempty"`
	AutoReset        bool   `json:"autoReset,omitempty"`
	Period           int64  `json:"period,omitempty"`
	AuthString       string `json:"auth_str,omitempty"`
	ProbeDestination string `json:"probeDestination,omitempty"`
}

type peerCredential struct {
	inboundID uint
	identity  string
}

func (user peerUser) credential() peerCredential {
	return peerCredential{user.InboundID, peerCredentialIdentity(user)}
}

func (user peerUser) valid() bool {
	if user.Period < 0 || user.Period > maxPeerPeriod || (user.DelayStart || user.AutoReset) != (user.Period > 0) {
		return false
	}
	if user.Expiry < 0 || user.Volume < 0 || (user.Speed != 0 && (user.Download != 0 || user.Upload != 0)) {
		return false
	}
	if user.InboundID == 0 || user.ClientID == 0 || strings.TrimSpace(user.Name) == "" || len(user.Name) > 256 || user.LimitIP < 0 || (user.IPLimitMode != "" && user.IPLimitMode != ipmonitor.ModeMonitor && user.IPLimitMode != ipmonitor.ModeEnforce) || (user.LimitIP > 0 && user.IPLimitMode == "") {
		return false
	}
	return validPeerProtocolCredential(user)
}

const maxPeerPeriod = 36525 * 86400

type PeerSyncResponse struct {
	Users           []peerUser       `json:"users"`
	InventoryDigest string           `json:"inventoryDigest"`
	Revision        string           `json:"revision"`
	Blocked         int              `json:"blocked"`
	Detached        bool             `json:"detached,omitempty"`
	Lease           *peerLease       `json:"lease,omitempty"`
	Probe           *peerProbeAccess `json:"probe,omitempty"`
	legacyInventory bool
	nextExpiry      int64
}

func (output *PeerSyncResponse) seal() error {
	encoded, err := json.Marshal(output.Users) // #nosec G117 -- Serialized credentials are used only as input to the revision digest.
	if err != nil {
		return err
	}
	output.Revision = nodeSecretDigest(output.InventoryDigest + string(encoded))
	return nil
}

func peerAssignments(db *gorm.DB, nodeUUID string) *gorm.DB {
	return db.Model(&model.PeerAssignment{}).Select("inbound_id, client_id").Where("node_uuid = ?", nodeUUID)
}

func peerUsersForAssignments(db, assignments *gorm.DB, inbounds []PeerInbound, now time.Time, legacy bool) (PeerSyncResponse, error) {
	return buildPeerUsersForAssignments(db, assignments, inbounds, now, legacy, "")
}

func buildPeerUsersForAssignments(db, assignments *gorm.DB, inbounds []PeerInbound, now time.Time, legacy bool, inventoryDigest string) (PeerSyncResponse, error) {
	output := PeerSyncResponse{Users: []peerUser{}, InventoryDigest: inventoryDigest}
	types := make(map[uint]string, len(inbounds))
	for _, inbound := range inbounds {
		if inbound.Link != nil {
			types[inbound.ID] = inbound.Type
		}
	}
	var rows []struct {
		InboundID uint
		model.Client
		Limits model.ClientLimits `gorm:"embedded;embeddedPrefix:lim_"`
	}
	err := peerAssignmentRows(db, assignments).Scan(&rows).Error
	if err != nil {
		return output, err
	}
	if len(rows) > maxPeerUsers {
		return output, errors.New("too many assigned users on this Slave")
	}
	credentials := make(map[peerCredential]int, len(rows))
	for _, row := range rows {
		delayStart := row.DelayStart || row.PeerDelayStart
		expiry := row.Expiry
		if delayStart && !row.AutoReset {
			expiry = 0
		}
		if expiry > 0 {
			if expiry <= now.Unix() {
				continue
			}
			if output.nextExpiry == 0 || expiry < output.nextExpiry {
				output.nextExpiry = expiry
			}
		}
		inboundType := types[row.InboundID]
		if legacy && (delayStart || row.AutoReset || !legacyPeerProtocol(inboundType)) {
			output.Blocked++
			continue
		}
		var protocols map[string]json.RawMessage
		var user peerUser
		limits := row.Limits
		if json.Unmarshal(row.Config, &protocols) != nil || json.Unmarshal(protocols[inboundType], &user) != nil {
			output.Blocked++
			continue
		}
		user.Type, user.InboundID, user.ClientID, user.Name = inboundType, row.InboundID, row.Id, row.Name
		user.LimitIP, user.IPLimitMode = row.LimitIP, row.IPLimitMode
		user.Expiry = expiry
		user.DelayStart, user.AutoReset, user.Period = delayStart, row.AutoReset, 0
		if user.DelayStart || user.AutoReset {
			user.Period = min(clientResetPeriodDays(row.ResetDays), maxPeerPeriod/86400) * 86400
		}
		user.Connections = limits.Connections
		user.Volume = row.Volume
		user.Speed, user.Download, user.Upload = 0, 0, 0
		if limits.SpeedShared {
			user.Speed = limits.Speed
		} else {
			user.Download, user.Upload = limits.DownloadSpeed, limits.UploadSpeed
		}
		if !user.valid() {
			output.Blocked++
			continue
		}
		credentials[user.credential()]++
		output.Users = append(output.Users, user)
	}
	output.Users = slices.DeleteFunc(output.Users, func(user peerUser) bool {
		if credentials[user.credential()] <= 1 {
			return false
		}
		output.Blocked++
		return true
	})
	err = output.seal()
	return output, err
}

func peerAssignmentRows(db, assignments *gorm.DB) *gorm.DB {
	return db.Table("(?) AS assignments", assignments).
		Select(`assignments.inbound_id, clients.id, clients.name, clients.config, clients.expiry,
			clients.delay_start, clients.peer_delay_start, clients.auto_reset, clients.reset_days,
			clients.limit_ip, clients.ip_limit_mode, clients.volume, limits.speed_shared AS lim_speed_shared, limits.speed AS lim_speed,
			limits.download_speed AS lim_download_speed, limits.upload_speed AS lim_upload_speed, limits.connections AS lim_connections`).
		Joins("JOIN clients ON clients.id = assignments.client_id").
		Joins("LEFT JOIN client_limits AS limits ON limits.client_id = clients.id").
		Where("clients.enable").
		Order("assignments.inbound_id, clients.id").Limit(maxPeerUsers + 1)
}

func (output *PeerSyncResponse) bindInventory(encoded []byte, host string) error {
	output.InventoryDigest = peerInventoryDigest(encoded, host)
	return output.seal()
}

func peerInventoryDigest(encoded []byte, host string) string {
	return nodeSecretDigest(host + "\x00" + string(encoded))
}

func managedInboundUsers(db *gorm.DB, inboundID uint, inboundType string, inbound map[string]any) ([]json.RawMessage, error) {
	if !peerProtocolSupported(inboundType) {
		return nil, nil
	}
	panel, err := readNodePanel(db)
	if err != nil || panel.Role != "slave" || len(panel.ManagedUsers) == 0 {
		return nil, err
	}
	var users []peerUser
	if err := json.Unmarshal(panel.ManagedUsers, &users); err != nil {
		return nil, err
	}
	var result []json.RawMessage
	for _, user := range users {
		if user.InboundID != inboundID || user.Type != inboundType {
			continue
		}
		encoded, err := json.Marshal(peerInboundCredential(user, inbound))
		if err != nil {
			return nil, err
		}
		result = append(result, encoded)
	}
	return result, nil
}

func (s *NodeService) applyPeerState(panel model.NodePanel, desired PeerSyncResponse) error {
	checksum := desired.Revision
	if err := desired.seal(); err != nil || checksum != desired.Revision || desired.Users == nil || len(desired.Users) > maxPeerUsers || desired.Blocked < 0 {
		return errors.New("invalid access revision from Master")
	}
	seen := make(map[[2]uint]bool, len(desired.Users))
	credentials := make(map[peerCredential]bool, len(desired.Users))
	for _, user := range desired.Users {
		key := [2]uint{user.InboundID, user.ClientID}
		credential := user.credential()
		if !user.valid() || user.ProbeDestination != "" || seen[key] || credentials[credential] {
			return errors.New("invalid user in access revision from Master")
		}
		seen[key] = true
		credentials[credential] = true
	}
	if probe := desired.Probe; probe != nil {
		address, err := netip.ParseAddrPort(probe.Destination)
		user := probe.user()
		now := time.Now()
		if err != nil || address.Port() == 0 || address.Addr().Zone() != "" || ssrf.IsInfrastructureAddr(address.Addr()) ||
			len(probe.ID) != 26 || probe.Until <= now.Unix() || probe.Until > now.Add(model.PeerAccessLease).Unix() ||
			!user.valid() || credentials[user.credential()] {
			return errors.New("invalid traffic check from Master")
		}
		for _, current := range desired.Users {
			if current.Name == user.Name {
				return errors.New("invalid traffic check identity from Master")
			}
		}
		desired.Users = append(desired.Users, user)
	}
	encoded, err := json.Marshal(desired.Users) // #nosec G117 -- The Slave's local proxy needs these credentials to authenticate clients.
	if err != nil {
		return err
	}
	if lease := desired.Lease; lease != nil && (lease.Until <= time.Now().Unix() || len(lease.Proof) != 64) {
		return errors.New("invalid access lease from Master")
	}
	return s.changeAppliedPeerState(panel.UUID, desired.Revision, desired.InventoryDigest, encoded, desired.Lease, desired.legacyInventory, false)
}

func hasNoRegularPeerUsers(data json.RawMessage) bool {
	var users []peerUser
	if err := json.Unmarshal(data, &users); err != nil || users == nil {
		return false
	}
	for _, user := range users {
		if user.ProbeDestination == "" {
			return false
		}
	}
	return true
}

func (s *NodeService) changeAppliedPeerState(identity, revision, inventoryDigest string, users []byte, lease *peerLease, legacyInventory, unlink bool) error {
	db := s.db()
	runtime := s.runtime()
	config := NewConfigServiceWithRuntime(runtime)
	return runtime.restart().runBlocking(func() error {
		instance := config.coreInstance()
		running := instance != nil && instance.IsRunning()
		if !running && !unlink {
			return errors.New("start the local proxy before applying access from Master")
		}
		var previous *[]byte
		stopped := false
		var replaced []peerProbeInboundChange
		err := db.Transaction(func(tx *gorm.DB) error {
			panel, err := readNodePanel(tx)
			if err != nil {
				return err
			}
			if panel.Role != "slave" || panel.UUID != identity {
				return errors.New("the local Slave pairing changed")
			}
			if unlink && revision != "" && (panel.AppliedRevision != revision || !hasNoRegularPeerUsers(panel.ManagedUsers)) {
				return errors.New("the local Slave has not applied the disconnect revision")
			}
			if !unlink {
				inbounds, err := announcePeerInbounds(tx)
				if err != nil {
					return err
				}
				if legacyInventory {
					inbounds = legacyPeerInventory(inbounds)
				}
				encoded, err := encodePeerInventory(inbounds)
				if err != nil {
					return err
				}
				if inventoryDigest != nodeSecretDigest(panel.PublicHost+"\x00"+string(encoded)) {
					return ErrInvalidPeerInventory
				}
			}
			now := time.Now().Unix()
			fields := map[string]any{"last_sync": now, "peer_access_until": int64(0), "peer_lease_proof": ""}
			if lease != nil {
				if lease.Until <= now {
					return errors.New("access lease from Master expired before apply")
				}
				fields["peer_access_until"], fields["peer_lease_proof"] = lease.Until, lease.Proof
			}
			if !unlink && panel.AppliedRevision == revision && string(panel.ManagedUsers) == string(users) {
				return tx.Model(&panel).UpdateColumns(fields).Error
			}
			var probeOnly []uint
			if !unlink && panel.AppliedRevision == revision {
				probeOnly = probeOnlyInboundIDs(panel.ManagedUsers, users)
			}
			if running {
				previous, err = config.getConfig(tx, "")
				if err != nil {
					return err
				}
			}
			if unlink {
				err = tx.Delete(&panel).Error
			} else {
				err = tx.Exec(`INSERT INTO peer_traffics(node_uuid, client_id, used, reset_at)
					SELECT ?, json_extract(value, '$.clientId'), 0, ? + json_extract(value, '$.period')
					FROM json_each(?) WHERE json_extract(value, '$.autoReset') AND NOT COALESCE(json_extract(value, '$.delayStart'), 0)
					ON CONFLICT(node_uuid, client_id) DO UPDATE SET reset_at = excluded.reset_at WHERE peer_traffics.reset_at = 0`, identity, now, string(users)).Error
				if err == nil {
					fields["applied_revision"], fields["managed_users"] = revision, users
					err = tx.Model(&panel).UpdateColumns(fields).Error
				}
			}
			if err != nil || !running {
				return err
			}
			next, err := config.getConfig(tx, "")
			if err != nil {
				return err
			}
			if err := core.ValidateConfig(*next); err != nil {
				return errors.New("access from Master is incompatible with the local proxy configuration")
			}
			if len(probeOnly) > 0 {
				changes, err := peerProbeInboundChanges(tx, probeOnly, *previous, *next)
				if err != nil {
					return err
				}
				for _, change := range changes {
					replaced = append(replaced, change)
					if err := instance.RemoveInbound(change.tag); err != nil {
						return errors.New("cannot reload traffic check inbound")
					}
					if err := instance.AddInbound(change.next); err != nil {
						return errors.New("cannot start traffic check inbound")
					}
				}
				return nil
			}
			stopped = true
			if err := instance.Stop(); err != nil {
				return errors.New("cannot stop the proxy to apply Master access")
			}
			if err := instance.Start(*next); err != nil {
				return errors.New("cannot start the proxy with the new Master access")
			}
			return nil
		})
		if err != nil && len(replaced) > 0 {
			for _, change := range slices.Backward(replaced) {
				_ = instance.RemoveInbound(change.tag)
				if restoreErr := instance.AddInbound(change.previous); restoreErr != nil {
					stopped = true
					break
				}
			}
		}
		if err != nil && stopped {
			if instance.IsRunning() {
				_ = instance.Stop()
			}
			if restoreErr := instance.Start(*previous); restoreErr != nil {
				logger.Error("cannot restore the proxy after applying Master access")
				return errors.Join(err, errors.New("previous proxy configuration could not be restarted"))
			}
		}
		if err == nil && (stopped || unlink || len(replaced) > 0) {
			ipmonitor.InvalidateAllCache()
		}
		return err
	})
}
