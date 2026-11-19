package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

const maxPeerUsers = 4096

type peerUser struct {
	InboundID   uint   `json:"inboundId"`
	ClientID    uint   `json:"clientId"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	UUID        string `json:"uuid,omitempty"`
	Password    string `json:"password,omitempty"`
	Flow        string `json:"flow,omitempty"`
	Expiry      int64  `json:"expiry,omitempty"`
	Connections uint32 `json:"connections,omitempty"`
	Speed       uint64 `json:"speed,omitempty"`
	Download    uint64 `json:"downloadSpeed,omitempty"`
	Upload      uint64 `json:"uploadSpeed,omitempty"`
	Volume      int64  `json:"volume,omitempty"`
	LimitIP     int    `json:"limitIp,omitempty"`
	IPLimitMode string `json:"ipLimitMode,omitempty"`
}

func (user peerUser) valid() bool {
	if user.Expiry < 0 || user.Volume < 0 || (user.Speed != 0 && (user.Download != 0 || user.Upload != 0)) {
		return false
	}
	if user.InboundID == 0 || user.ClientID == 0 || strings.TrimSpace(user.Name) == "" || len(user.Name) > 256 || user.LimitIP < 0 || (user.IPLimitMode != "" && user.IPLimitMode != ipmonitor.ModeMonitor && user.IPLimitMode != ipmonitor.ModeEnforce) || (user.LimitIP > 0 && user.IPLimitMode == "") {
		return false
	}
	switch user.Type {
	case "vless", "vmess":
		id, err := uuid.FromString(user.UUID)
		return err == nil && id != uuid.Nil && user.Password == "" && (user.Flow == "" || user.Type == "vless" && user.Flow == "xtls-rprx-vision")
	case "trojan":
		return user.Password != "" && len(user.Password) <= 256 && user.UUID == "" && user.Flow == ""
	}
	return false
}

type PeerSyncResponse struct {
	Users           []peerUser `json:"users"`
	InventoryDigest string     `json:"inventoryDigest"`
	Revision        string     `json:"revision"`
	Blocked         int        `json:"blocked"`
	Detached        bool       `json:"detached,omitempty"`
}

func (output *PeerSyncResponse) seal() error {
	encoded, err := json.Marshal(output.Users) // #nosec G117 -- Serialized credentials are used only as input to the revision digest.
	if err != nil {
		return err
	}
	output.Revision = nodeSecretDigest(output.InventoryDigest + string(encoded))
	return nil
}

func desiredPeerUsers(db *gorm.DB, nodeUUID string, inbounds []PeerInbound) (PeerSyncResponse, error) {
	output := PeerSyncResponse{Users: []peerUser{}}
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
	err := db.Table("peer_assignments AS assignments").
		Select(`assignments.inbound_id, clients.*, limits.speed_shared AS lim_speed_shared, limits.speed AS lim_speed,
			limits.download_speed AS lim_download_speed, limits.upload_speed AS lim_upload_speed, limits.connections AS lim_connections`).
		Joins("JOIN clients ON clients.id = assignments.client_id").
		Joins("LEFT JOIN client_limits AS limits ON limits.client_id = clients.id").
		Where("assignments.node_uuid = ? AND clients.enable", nodeUUID).
		Order("assignments.inbound_id, clients.id").Limit(maxPeerUsers + 1).Scan(&rows).Error
	if err != nil {
		return output, err
	}
	if len(rows) > maxPeerUsers {
		return output, errors.New("too many assigned users on this Slave")
	}
	now := time.Now().Unix()
	for _, row := range rows {
		if row.Expiry > 0 && row.Expiry <= now {
			continue
		}
		inboundType := types[row.InboundID]
		var protocols map[string]json.RawMessage
		var user peerUser
		limits := row.Limits
		if row.DelayStart || row.AutoReset ||
			json.Unmarshal(row.Config, &protocols) != nil || json.Unmarshal(protocols[inboundType], &user) != nil {
			output.Blocked++
			continue
		}
		user.Type, user.InboundID, user.ClientID, user.Name = inboundType, row.InboundID, row.Id, row.Name
		user.LimitIP, user.IPLimitMode = row.LimitIP, row.IPLimitMode
		user.Expiry = row.Expiry
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
		output.Users = append(output.Users, user)
	}
	err = output.seal()
	return output, err
}

func (output *PeerSyncResponse) bindInventory(encoded []byte, host string) error {
	output.InventoryDigest = nodeSecretDigest(host + "\x00" + string(encoded))
	return output.seal()
}

func managedInboundUsers(db *gorm.DB, inboundID uint, inboundType string, inbound map[string]any) ([]json.RawMessage, error) {
	if inboundType != "vless" && inboundType != "vmess" && inboundType != "trojan" {
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
		entry := map[string]string{"name": user.Name}
		if user.Type == "trojan" {
			entry["password"] = user.Password
		} else {
			entry["uuid"] = user.UUID
			transport, _ := inbound["transport"].(map[string]any)
			transportType, _ := transport["type"].(string)
			tls, _ := inbound["tls"].(map[string]any)
			if user.Flow != "" && tls["enabled"] == true && (transportType == "" || transportType == "tcp") {
				entry["flow"] = user.Flow
			}
		}
		encoded, err := json.Marshal(entry)
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
	for _, user := range desired.Users {
		key := [2]uint{user.InboundID, user.ClientID}
		if !user.valid() || seen[key] {
			return errors.New("invalid user in access revision from Master")
		}
		seen[key] = true
	}
	encoded, err := json.Marshal(desired.Users) // #nosec G117 -- The Slave's local proxy needs these credentials to authenticate clients.
	if err != nil {
		return err
	}
	return s.changeAppliedPeerState(panel.UUID, desired.Revision, desired.InventoryDigest, encoded, false)
}

func (s *NodeService) changeAppliedPeerState(identity, revision, inventoryDigest string, users []byte, unlink bool) error {
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
		err := db.Transaction(func(tx *gorm.DB) error {
			panel, err := readNodePanel(tx)
			if err != nil {
				return err
			}
			if panel.Role != "slave" || panel.UUID != identity {
				return errors.New("the local Slave pairing changed")
			}
			if unlink && revision != "" && (panel.AppliedRevision != revision || string(panel.ManagedUsers) != "[]") {
				return errors.New("the local Slave has not applied the disconnect revision")
			}
			if !unlink {
				inbounds, err := announcePeerInbounds(tx)
				if err != nil {
					return err
				}
				encoded, err := encodePeerInventory(inbounds)
				if err != nil {
					return err
				}
				if inventoryDigest != nodeSecretDigest(panel.PublicHost+"\x00"+string(encoded)) {
					return ErrInvalidPeerInventory
				}
			}
			if !unlink && panel.AppliedRevision == revision && string(panel.ManagedUsers) == string(users) {
				return nil
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
				err = tx.Model(&panel).UpdateColumns(map[string]any{"applied_revision": revision, "managed_users": users}).Error
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
			stopped = true
			if err := instance.Stop(); err != nil {
				return errors.New("cannot stop the proxy to apply Master access")
			}
			if err := instance.Start(*next); err != nil {
				return errors.New("cannot start the proxy with the new Master access")
			}
			return nil
		})
		if err != nil && stopped {
			if instance.IsRunning() {
				_ = instance.Stop()
			}
			if restoreErr := instance.Start(*previous); restoreErr != nil {
				logger.Error("cannot restore the proxy after applying Master access")
				return errors.Join(err, errors.New("previous proxy configuration could not be restarted"))
			}
		}
		if err == nil && (stopped || unlink) {
			ipmonitor.InvalidateAllCache()
		}
		return err
	})
}
