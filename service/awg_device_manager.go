package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.org/x/text/unicode/norm"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gorm.io/gorm"
)

const (
	awgMaxDeviceNameRunes = 64
	awgMaxRequestKeyRunes = 128
)

var (
	ErrAWGClientInactive      = errors.New("AWG client is inactive")
	ErrAWGDeviceLimitReached  = errors.New("AWG device limit reached")
	ErrAWGInvalidDeviceName   = errors.New("invalid AWG device name")
	ErrAWGInvalidRequestKey   = errors.New("invalid AWG create request key")
	ErrAWGIdempotencyConflict = errors.New("AWG create request conflicts with existing device")
	ErrAWGProvisioningFailed  = errors.New("AWG provisioning failed")
)

type AWGDeviceInfo struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	PublicKey      string `json:"publicKey"`
	IPv4Address    string `json:"ipv4Address"`
	DesiredEnabled bool   `json:"desiredEnabled"`
	SyncState      string `json:"syncState"`
	Provisioned    bool   `json:"provisioned"`
	LastHandshake  int64  `json:"lastHandshake"`
	TotalRx        uint64 `json:"totalRx"`
	TotalTx        uint64 `json:"totalTx"`
	CreatedAt      int64  `json:"createdAt"`
}

type AWGGeneratedKeys struct {
	PrivateKey []byte
	PublicKey  []byte
	PSK        []byte
}

type awgKeyCipher interface {
	Encrypt(clientID uint, cryptoContext, plaintext []byte) ([]byte, error)
	Decrypt(clientID uint, cryptoContext, ciphertext []byte) ([]byte, error)
}

type AWGManagerDeps struct {
	DB                *gorm.DB
	LoadSettings      func() (AWGSettings, error)
	LoadEndpoint      func(*gorm.DB, AWGSettings) (AWGManagedEndpoint, error)
	SyncEndpointPeers func(*gorm.DB, AWGSettings, []AWGPersistedPeer) error
	GenerateKeys      func() (AWGGeneratedKeys, error)
	NewCryptoContext  func() ([]byte, error)
	Cipher            awgKeyCipher
	CipherFactory     func() (awgKeyCipher, error)
	Now               func() int64
}

func defaultAWGManagerDeps() AWGManagerDeps {
	return AWGManagerDeps{
		DB: database.GetDB(),
		LoadSettings: func() (AWGSettings, error) {
			return (&SettingService{}).GetAWGSettings()
		},
		LoadEndpoint:      LoadAWGManagedEndpoint,
		SyncEndpointPeers: SyncAWGManagedEndpointPeers,
		GenerateKeys:      generateAWGKeys,
		NewCryptoContext:  NewAWGCryptoContext,
		CipherFactory: func() (awgKeyCipher, error) {
			return NewAWGCipherFromEnv()
		},
		Now: func() int64 { return time.Now().Unix() },
	}
}

func generateAWGKeys() (AWGGeneratedKeys, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return AWGGeneratedKeys{}, fmt.Errorf("generate AWG private key: %w", err)
	}
	psk, err := wgtypes.GenerateKey()
	if err != nil {
		return AWGGeneratedKeys{}, fmt.Errorf("generate AWG preshared key: %w", err)
	}
	publicKey := privateKey.PublicKey()
	return AWGGeneratedKeys{
		PrivateKey: append([]byte(nil), privateKey[:]...),
		PublicKey:  append([]byte(nil), publicKey[:]...),
		PSK:        append([]byte(nil), psk[:]...),
	}, nil
}

func (m *AWGManager) CreateDevice(ctx context.Context, clientID uint, requestKey, name string, effectiveLimit int) (AWGDeviceInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AWGDeviceInfo{}, err
	}
	requestKey, err := normalizeAWGRequestKey(requestKey)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	name, err = normalizeAWGDeviceName(name)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	if m == nil || m.deps.DB == nil {
		return AWGDeviceInfo{}, errors.New("AWG database is unavailable")
	}

	result := m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		info, createErr := m.createDeviceInWorker(operationCtx, clientID, requestKey, name, effectiveLimit)
		return awgCommandResult{device: info, err: createErr}
	})
	return result.device, result.err
}

func (m *AWGManager) createDeviceInWorker(ctx context.Context, clientID uint, requestKey, name string, effectiveLimit int) (AWGDeviceInfo, error) {
	now := m.deps.Now()
	var replay model.AWGDevice
	replayErr := m.deps.DB.Where("client_id = ? AND create_request_key = ?", clientID, requestKey).First(&replay).Error
	if replayErr == nil {
		if replay.Name != name || !replay.DesiredEnabled || (replay.SyncState != "pending_add" && replay.SyncState != "in_sync") {
			return AWGDeviceInfo{}, ErrAWGIdempotencyConflict
		}
		if replay.SyncState == "in_sync" && replay.Provisioned {
			return awgDeviceInfo(replay), nil
		}
	} else if !errors.Is(replayErr, gorm.ErrRecordNotFound) {
		return AWGDeviceInfo{}, replayErr
	}
	settings, err := m.deps.LoadSettings()
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	if !settings.Enabled {
		return AWGDeviceInfo{}, errors.New("AWG is disabled")
	}
	endpoint, err := m.deps.LoadEndpoint(m.deps.DB, settings)
	if err != nil {
		return AWGDeviceInfo{}, err
	}

	device := replay
	created := false
	err = m.deps.DB.Transaction(func(tx *gorm.DB) error {
		lookupErr := tx.Where("client_id = ? AND create_request_key = ?", clientID, requestKey).First(&device).Error
		if lookupErr == nil {
			if device.Name != name || !device.DesiredEnabled || (device.SyncState != "pending_add" && device.SyncState != "in_sync") {
				return ErrAWGIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}
		var client model.Client
		if findErr := tx.First(&client, clientID).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return ErrAWGClientInactive
			}
			return findErr
		}
		if !clientIsActiveAt(client, now) {
			return ErrAWGClientInactive
		}
		if effectiveLimit <= 0 {
			return ErrAWGDeviceLimitReached
		}
		var count int64
		if countErr := tx.Model(&model.AWGDevice{}).Where("client_id = ? AND desired_enabled = ?", clientID, true).Count(&count).Error; countErr != nil {
			return countErr
		}
		if count >= int64(effectiveLimit) {
			return ErrAWGDeviceLimitReached
		}
		keys, keyErr := m.deps.GenerateKeys()
		if keyErr != nil {
			return keyErr
		}
		defer clear(keys.PrivateKey)
		defer clear(keys.PSK)
		if len(keys.PrivateKey) != 32 || len(keys.PublicKey) != 32 || len(keys.PSK) != 32 {
			return errors.New("generated AWG key material is invalid")
		}
		cryptoContext, contextErr := m.deps.NewCryptoContext()
		if contextErr != nil {
			return contextErr
		}
		cipher, cipherErr := m.awgCipher()
		if cipherErr != nil {
			return cipherErr
		}
		privateEnc, cipherErr := cipher.Encrypt(clientID, cryptoContext, keys.PrivateKey)
		if cipherErr != nil {
			return cipherErr
		}
		pskEnc, cipherErr := cipher.Encrypt(clientID, cryptoContext, keys.PSK)
		if cipherErr != nil {
			return cipherErr
		}
		address, allocErr := AllocateAWGIPv4(tx, settings.Subnet, endpoint.ServerAddress, now)
		if allocErr != nil {
			return allocErr
		}
		device = model.AWGDevice{ClientId: clientID, Name: name, CreateRequestKey: requestKey, CryptoContext: cryptoContext,
			PublicKey: base64.StdEncoding.EncodeToString(keys.PublicKey), PrivateKeyEnc: privateEnc, PSKEnc: pskEnc,
			IPv4Address: address.String(), DesiredEnabled: true, SyncState: "pending_add", CreatedAt: now, UpdatedAt: now}
		created = true
		return tx.Create(&device).Error
	})
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	if !created && device.SyncState == "in_sync" && device.Provisioned {
		return awgDeviceInfo(device), nil
	}
	if !created {
		var client model.Client
		if err := m.deps.DB.First(&client, clientID).Error; err != nil || !clientIsActiveAt(client, now) {
			return AWGDeviceInfo{}, ErrAWGClientInactive
		}
		if effectiveLimit <= 0 {
			return AWGDeviceInfo{}, ErrAWGDeviceLimitReached
		}
	}
	cipher, err := m.awgCipher()
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	psk, err := cipher.Decrypt(clientID, device.CryptoContext, device.PSKEnc)
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	defer clear(psk)
	peer := AWGPeerSpec{PublicKey: device.PublicKey, PresharedKey: base64.StdEncoding.EncodeToString(psk), AllowedIP: netipPrefix32(device.IPv4Address)}
	provisionErr := func() error {
		if m.provisioner == nil {
			return ErrAWGProvisionerMissing
		}
		return m.provisioner.Add(ctx, peer)
	}()
	if provisionErr != nil {
		_ = m.deps.DB.Model(&model.AWGDevice{}).
			Where("id = ? AND sync_state = ? AND desired_enabled = ?", device.Id, "pending_add", true).
			Updates(map[string]any{"last_error": "AWG provisioning failed", "updated_at": m.deps.Now()}).Error
		return AWGDeviceInfo{}, ErrAWGProvisioningFailed
	}
	updateResult := m.deps.DB.Model(&model.AWGDevice{}).
		Where("id = ? AND sync_state = ? AND desired_enabled = ? AND public_key = ?", device.Id, "pending_add", true, device.PublicKey).
		Updates(map[string]any{"sync_state": "in_sync", "provisioned": true, "last_error": "", "updated_at": m.deps.Now()})
	if updateResult.Error != nil {
		return AWGDeviceInfo{}, updateResult.Error
	}
	if updateResult.RowsAffected != 1 {
		return AWGDeviceInfo{}, errors.New("AWG create state changed during provisioning")
	}
	device.SyncState, device.Provisioned, device.LastError = "in_sync", true, ""
	return awgDeviceInfo(device), nil
}

func (m *AWGManager) awgCipher() (awgKeyCipher, error) {
	if m.deps.Cipher != nil {
		return m.deps.Cipher, nil
	}
	if m.deps.CipherFactory != nil {
		return m.deps.CipherFactory()
	}
	return nil, ErrAWGEncryptionUnavailable
}

func normalizeAWGDeviceName(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", ErrAWGInvalidDeviceName
	}
	value = norm.NFC.String(value)
	for _, r := range value {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return "", ErrAWGInvalidDeviceName
		}
	}
	value = strings.Join(strings.Fields(value), " ")
	if value == "" || utf8.RuneCountInString(value) > awgMaxDeviceNameRunes {
		return "", ErrAWGInvalidDeviceName
	}
	return value, nil
}

func normalizeAWGRequestKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > awgMaxRequestKeyRunes {
		return "", ErrAWGInvalidRequestKey
	}
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:-", r)) {
			return "", ErrAWGInvalidRequestKey
		}
	}
	return value, nil
}

func awgDeviceInfo(device model.AWGDevice) AWGDeviceInfo {
	return AWGDeviceInfo{
		ID: device.Id, Name: device.Name, PublicKey: device.PublicKey, IPv4Address: device.IPv4Address,
		DesiredEnabled: device.DesiredEnabled, SyncState: device.SyncState, Provisioned: device.Provisioned,
		LastHandshake: device.LastHandshake, TotalRx: device.TotalRx, TotalTx: device.TotalTx, CreatedAt: device.CreatedAt,
	}
}

func netipPrefix32(value string) netip.Prefix {
	address, _ := netip.ParseAddr(value)
	return netip.PrefixFrom(address, 32)
}
