package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

type PeerInbound struct {
	ID   uint            `json:"id"`
	Tag  string          `json:"tag"`
	Type string          `json:"type"`
	Link *peerPublicLink `json:"link,omitempty"`
}

type PeerSyncRequest struct {
	Inbounds        []PeerInbound `json:"inbounds"`
	PublicHost      string        `json:"publicHost"`
	AppliedRevision string        `json:"appliedRevision,omitempty"`
}

var ErrInvalidPeerInventory = errors.New("invalid peer inventory")

func encodePeerInventory(inbounds []PeerInbound) ([]byte, error) {
	inventory := NodeInventory{}
	for _, inbound := range inbounds {
		inventory.Inbounds = append(inventory.Inbounds, NodeInventoryInbound{ID: inbound.ID, Tag: inbound.Tag, Type: inbound.Type, PeerLink: inbound.Link})
	}
	return json.Marshal(inventory)
}

var hostnameLabel = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func validPeerHost(host string) bool {
	if host == "" || net.ParseIP(host) != nil {
		return true
	}
	if len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if !hostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}

func (s *NodeService) SetSlavePublicHost(host string) error {
	host = strings.TrimSpace(host)
	if !validPeerHost(host) {
		return errors.New("enter a hostname or IP address without a scheme or port")
	}
	db := s.db()
	if db == nil {
		return errors.New("database is not initialized")
	}
	updated := db.Model(&model.NodePanel{}).Where("id = ? AND role = ?", 1, "slave").UpdateColumn("public_host", host)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("this setting is only available on Slave")
	}
	return nil
}

func (s *NodeService) AcceptPeerSync(identity, token string, input PeerSyncRequest) (PeerSyncResponse, error) {
	db := s.db()
	panel, err := readNodePanel(db)
	if err != nil {
		return PeerSyncResponse{}, err
	}
	if panel.Role != "master" {
		return PeerSyncResponse{}, ErrInvalidNodeCredential
	}
	var desired PeerSyncResponse
	var invalidate bool
	err = db.Transaction(func(tx *gorm.DB) error {
		node, err := enrolledNode(tx, identity, token)
		if errors.Is(err, ErrInvalidNodeCredential) && len(identity) == 36 && len(token) >= 26 && len(token) <= 64 && len(input.AppliedRevision) == 64 {
			var receipt model.NodeDetachReceipt
			lookup := tx.Where("uuid = ? AND digest = ? AND revision = ? AND expires_at > ?", identity, nodeSecretDigest(token), input.AppliedRevision, time.Now().Unix()).Take(&receipt).Error
			if lookup == nil {
				desired = PeerSyncResponse{Users: []peerUser{}, Revision: receipt.Revision, Detached: true}
				return nil
			}
			if !errors.Is(lookup, gorm.ErrRecordNotFound) {
				return lookup
			}
		}
		if err != nil {
			return err
		}
		if len(input.Inbounds) > 512 || !validPeerHost(input.PublicHost) {
			return ErrInvalidPeerInventory
		}
		seen := make(map[uint]bool, len(input.Inbounds))
		ids := make([]uint, 0, len(input.Inbounds))
		for _, inbound := range input.Inbounds {
			if inbound.ID == 0 || seen[inbound.ID] || inbound.Tag == "" || len(inbound.Tag) > 256 || inbound.Type == "" || len(inbound.Type) > 64 {
				return ErrInvalidPeerInventory
			}
			seen[inbound.ID] = true
			ids = append(ids, inbound.ID)
			if inbound.Link != nil && !inbound.Link.valid() {
				return ErrInvalidPeerInventory
			}
		}
		encoded, err := encodePeerInventory(input.Inbounds)
		if err != nil {
			return err
		}
		if node.PeerDisconnect {
			desired = PeerSyncResponse{Users: []peerUser{}}
		} else {
			desired, err = desiredPeerUsers(tx, node.UUID, input.Inbounds)
			if err != nil {
				return err
			}
		}
		if err := desired.bindInventory(encoded, input.PublicHost); err != nil {
			return err
		}
		acknowledged := ""
		if input.AppliedRevision == desired.Revision {
			acknowledged = input.AppliedRevision
			if node.PeerDisconnect {
				now := time.Now()
				if err := tx.Where("expires_at <= ?", now.Unix()).Delete(&model.NodeDetachReceipt{}).Error; err != nil {
					return err
				}
				receipt := model.NodeDetachReceipt{UUID: node.UUID, Digest: node.EnrollmentDigest, Revision: desired.Revision, ExpiresAt: now.Add(24 * time.Hour).Unix()}
				if err := tx.Create(&receipt).Error; err != nil {
					return err
				}
				desired = PeerSyncResponse{Users: []peerUser{}, Revision: receipt.Revision, Detached: true}
				if err := tx.Where("node_uuid = ?", node.UUID).Delete(&model.PeerAssignment{}).Error; err != nil {
					return err
				}
				return tx.Delete(node).Error
			}
		}
		invalidate = node.PeerRevision != acknowledged || node.LinkHost != input.PublicHost || !bytes.Equal(node.Inventory, encoded)
		updated := tx.Model(&model.Node{}).Where("id = ? AND enabled AND enrollment_digest = ?", node.Id, node.EnrollmentDigest).UpdateColumns(map[string]any{"last_peer_sync": time.Now().Unix(), "inventory": json.RawMessage(encoded), "link_host": input.PublicHost, "peer_revision": acknowledged, "peer_blocked": desired.Blocked})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrInvalidNodeCredential
		}
		stale := tx.Where("node_uuid = ?", node.UUID)
		if len(ids) > 0 {
			stale = stale.Where("inbound_id NOT IN ?", ids)
		}
		return stale.Delete(model.PeerAssignment{}).Error
	})
	if err == nil && invalidate && invalidateSubscriptionCacheAfterSave != nil {
		invalidateSubscriptionCacheAfterSave()
	}
	return desired, err
}

func (s *NodeService) SyncWithMaster(ctx context.Context) {
	db := s.db()
	panel, err := readNodePanel(db)
	if err != nil || panel.Role != "slave" || ctx.Err() != nil {
		return
	}
	now := time.Now().Unix()
	fields := map[string]any{"sync_error": ""}
	inbounds, err := announcePeerInbounds(db)
	if err != nil {
		err = errors.New("cannot read local inbounds")
	} else if len(inbounds) > 512 {
		err = ErrInvalidPeerInventory
	} else {
		var desired PeerSyncResponse
		var applied string
		if instance := s.runtime().Core(); instance != nil && instance.IsRunning() {
			applied = panel.AppliedRevision
		}
		desired, err = sendPeerSync(ctx, panel, PeerSyncRequest{Inbounds: inbounds, PublicHost: panel.PublicHost, AppliedRevision: applied})
		if err == nil && ctx.Err() == nil {
			if desired.Detached {
				if applied == "" || desired.Revision != applied || len(desired.Users) != 0 {
					err = errors.New("invalid disconnect receipt from Master")
				} else {
					err = s.changeAppliedPeerState(panel.UUID, applied, "", nil, true)
				}
			} else {
				err = s.applyPeerState(panel, desired)
			}
			if err == nil && desired.Blocked > 0 {
				fields["sync_error"] = "Some assignments cannot be applied: check the protocol, credentials and client limits on Master."
			}
		}
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		fields["sync_error"] = err.Error()
	} else {
		fields["last_sync"] = now
	}
	_ = db.Model(&model.NodePanel{}).Where("id = ? AND role = ? AND uuid = ?", panel.ID, "slave", panel.UUID).UpdateColumns(fields).Error
}

func sendPeerSync(ctx context.Context, panel model.NodePanel, input PeerSyncRequest) (PeerSyncResponse, error) {
	endpoint, err := nodeMasterEndpoint(panel.MasterURL, "sync")
	if err != nil {
		return PeerSyncResponse{}, err
	}
	token, err := (&SettingService{}).decryptSettingValue("nodePeerToken", panel.Token)
	if err != nil {
		return PeerSyncResponse{}, errors.New("cannot read the saved node credential")
	}
	payload, err := json.Marshal(input)
	if err != nil || len(payload) > 256<<10 {
		return PeerSyncResponse{}, ErrInvalidPeerInventory
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return PeerSyncResponse{}, errors.New("invalid sync URL")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Node-ID", panel.UUID)
	response, err := nodePeerHTTPClient.Do(request)
	if err != nil {
		return PeerSyncResponse{}, errors.New("cannot reach Master over HTTPS; check the address and certificate")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return PeerSyncResponse{}, errors.New("the node credential is no longer accepted by Master")
	}
	if response.StatusCode != http.StatusOK {
		return PeerSyncResponse{}, errors.New("synchronization was not accepted by Master")
	}
	const maxResponseBytes = 4 << 20
	reader := &io.LimitedReader{R: response.Body, N: maxResponseBytes + 1}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var output *PeerSyncResponse
	if err := decoder.Decode(&output); err != nil || output == nil || decoder.Decode(new(any)) != io.EOF || reader.N == 0 {
		return PeerSyncResponse{}, errors.New("invalid sync response from Master")
	}
	return *output, nil
}
