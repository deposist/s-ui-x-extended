package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidNodeInvite     = errors.New("node invitation is invalid or expired")
	ErrInvalidNodeCredential = errors.New("node credential is invalid or revoked")
	ErrSlaveHasLocalData     = errors.New("move local clients and disconnect child nodes before switching to Slave")
	ErrSlaveClientWrite      = errors.New("clients are managed on the Master panel")
)

const nodeInviteLifetime = 10 * time.Minute

func readNodePanel(db *gorm.DB) (model.NodePanel, error) {
	if db == nil {
		return model.NodePanel{}, errors.New("database is not initialized")
	}
	var panel model.NodePanel
	err := db.Take(&panel, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.NodePanel{ID: 1, Role: "standalone"}, nil
	}
	return panel, err
}

func checkSlaveEligibility(db *gorm.DB) error {
	if db == nil {
		return errors.New("database is not initialized")
	}
	var client model.Client
	err := db.Select("id").Take(&client).Error
	if err == nil {
		return ErrSlaveHasLocalData
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var node model.Node
	err = db.Select("id").Where("enabled = ?", true).Take(&node).Error
	if err == nil {
		return ErrSlaveHasLocalData
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}

func RequireLocalClientWrites(db *gorm.DB) error {
	panel, err := readNodePanel(db)
	if err != nil {
		return err
	}
	if panel.Role == "slave" {
		return ErrSlaveClientWrite
	}
	return nil
}

func nodeMasterEndpoint(address, action string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(address))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" || strings.Contains(address, "#") {
		return "", errors.New("enter the HTTPS address of the Master panel without credentials, query or fragment")
	}
	return parsed.JoinPath("node-peer", action).String(), nil
}

type nodeEnrollment struct {
	UUID  string `json:"uuid"`
	Token string `json:"token"`
}

func (s *NodeService) Panel() (model.NodePanel, error) {
	return readNodePanel(s.db())
}

type NodePanelSetup struct {
	model.NodePanel
	CompatibleInbounds []PeerInbound `json:"compatibleInbounds,omitempty"`
}

func (s *NodeService) PanelSetup() (NodePanelSetup, error) {
	panel, err := s.Panel()
	setup := NodePanelSetup{NodePanel: panel}
	if err != nil || panel.Role != "slave" {
		return setup, err
	}
	inbounds, err := announcePeerInbounds(s.db())
	for _, inbound := range inbounds {
		if peerProbeEligible(inbound) {
			inbound.Link = nil
			setup.CompatibleInbounds = append(setup.CompatibleInbounds, inbound)
		}
	}
	return setup, err
}

func (s *NodeService) SetMaster() error {
	db := s.db()
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := RequireLocalClientWrites(tx); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.NodePanel{ID: 1, Role: "master"}).Error
	})
}

func (s *NodeService) UnlinkSlave() error {
	panel, err := s.Panel()
	if err != nil || panel.Role != "slave" {
		return err
	}
	return s.changeAppliedPeerState(panel.UUID, "", "", nil, nil, false, true)
}

func (s *NodeService) Invite() (string, error) {
	invite, err := s.InviteDetails()
	return invite.Code, err
}

type NodeInvitation struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expiresAt"`
}

func (s *NodeService) InviteDetails() (NodeInvitation, error) {
	panel, err := s.Panel()
	if err != nil {
		return NodeInvitation{}, err
	}
	if panel.Role != "master" {
		return NodeInvitation{}, errors.New("select Master before creating an invitation")
	}
	now := time.Now()
	code, err := createNodeInvite(s.db(), now)
	if err != nil {
		return NodeInvitation{}, err
	}
	return NodeInvitation{Code: code, ExpiresAt: now.Add(nodeInviteLifetime).Unix()}, nil
}

func (s *NodeService) RedeemInvite(code, name string) (nodeEnrollment, error) {
	db := s.db()
	panel, err := readNodePanel(db)
	if err != nil {
		return nodeEnrollment{}, err
	}
	if panel.Role != "master" {
		return nodeEnrollment{}, ErrInvalidNodeInvite
	}
	return redeemNodeInvite(db, time.Now(), code, name)
}

var nodePeerHTTPClient = http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func (s *NodeService) JoinMaster(ctx context.Context, address, name, code string) error {
	address, name, code = strings.TrimSpace(address), strings.TrimSpace(name), strings.TrimSpace(code)
	if name == "" {
		name, _ = os.Hostname()
		if name == "" {
			name = "Slave"
		}
	}
	endpoint, err := nodeMasterEndpoint(address, "enroll")
	if err != nil {
		return err
	}
	if len(name) == 0 || len(name) > 128 || len(code) < 26 || len(code) > 64 {
		return ErrInvalidNodeInvite
	}
	db := s.db()
	if err := checkSlaveEligibility(db); err != nil {
		return err
	}
	if err := RequireLocalClientWrites(db); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{"name": name, "code": code})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := nodePeerHTTPClient.Do(request)
	if err != nil {
		return errors.New("cannot reach Master over HTTPS; check the address and certificate")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return ErrInvalidNodeInvite
	}
	if response.StatusCode != http.StatusCreated {
		return errors.New("pairing was not accepted by Master at this address")
	}
	var credential nodeEnrollment
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&credential); err != nil {
		return errors.New("invalid pairing response from Master")
	}
	if _, err := uuid.FromString(credential.UUID); err != nil || len(credential.Token) < 26 || len(credential.Token) > 64 {
		return errors.New("invalid pairing response from Master")
	}
	token, err := (&SettingService{}).encryptSettingValue("nodePeerToken", credential.Token)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := checkSlaveEligibility(tx); err != nil {
			return err
		}
		if err := RequireLocalClientWrites(tx); err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.NodeInvite{}).Error; err != nil {
			return err
		}
		return tx.Save(&model.NodePanel{ID: 1, Role: "slave", MasterURL: address, UUID: credential.UUID, Token: token}).Error
	})
}

func nodeSecretDigest(secret string) string {
	digest := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(digest[:])
}

func createNodeInvite(db *gorm.DB, now time.Time) (string, error) {
	code := rand.Text()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ?", now.Unix()).Delete(&model.NodeInvite{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.NodeInvite{Digest: nodeSecretDigest(code), ExpiresAt: now.Add(nodeInviteLifetime).Unix()}).Error
	})
	if err != nil {
		return "", err
	}
	return code, nil
}

func redeemNodeInvite(db *gorm.DB, now time.Time, code, name string) (nodeEnrollment, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 || len(code) < 26 || len(code) > 64 {
		return nodeEnrollment{}, ErrInvalidNodeInvite
	}
	identity, err := uuid.NewV4()
	if err != nil {
		return nodeEnrollment{}, err
	}
	credential := nodeEnrollment{UUID: identity.String(), Token: rand.Text()}
	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("digest = ? AND expires_at > ?", nodeSecretDigest(code), now.Unix()).Delete(&model.NodeInvite{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidNodeInvite
		}
		return tx.Create(&model.Node{UUID: credential.UUID, Name: name, Enabled: true, EnrollmentDigest: nodeSecretDigest(credential.Token)}).Error
	})
	if err != nil {
		return nodeEnrollment{}, err
	}
	return credential, nil
}

func enrolledNode(db *gorm.DB, identity, token string) (*model.Node, error) {
	if len(identity) != 36 || len(token) < 26 || len(token) > 64 {
		return nil, ErrInvalidNodeCredential
	}
	var node model.Node
	err := db.Where("uuid = ? AND enabled = ? AND enrollment_digest = ?", identity, true, nodeSecretDigest(token)).Take(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidNodeCredential
	}
	if err != nil {
		return nil, err
	}
	return &node, nil
}
