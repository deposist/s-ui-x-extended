package service

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

var (
	ErrAWGOperationInProgress = errors.New("another AWG device operation is in progress")
	ErrAWGRotationFailed      = errors.New("AWG rotation failed")
)

// RotateOwnedDevice replaces an owned device's keys without ever restoring the
// old peer. A successful return means the old peer was verified absent and the
// new peer was verified present.
func (m *AWGManager) RotateOwnedDevice(ctx context.Context, deviceID, clientID uint, requestKey string) (AWGDeviceInfo, error) {
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
	if m == nil || m.deps.DB == nil {
		return AWGDeviceInfo{}, errors.New("AWG database is unavailable")
	}

	result := m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		info, rotateErr := m.rotateOwnedDeviceInWorker(operationCtx, deviceID, clientID, requestKey)
		return awgCommandResult{device: info, err: rotateErr}
	})
	return result.device, result.err
}

func (m *AWGManager) rotateOwnedDeviceInWorker(ctx context.Context, deviceID, clientID uint, requestKey string) (AWGDeviceInfo, error) {
	var device model.AWGDevice
	if err := m.deps.DB.Where("id = ? AND client_id = ?", deviceID, clientID).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AWGDeviceInfo{}, ErrAWGDeviceNotFound
		}
		return AWGDeviceInfo{}, err
	}
	if device.SyncState == "in_sync" && device.Provisioned && device.DesiredEnabled && device.RotateRequestKey == requestKey {
		return awgDeviceInfo(device), nil
	}
	if device.SyncState == "pending_rotate" {
		if device.RotateRequestKey != requestKey {
			return AWGDeviceInfo{}, ErrAWGOperationInProgress
		}
		return m.resumeAWGRotation(ctx, device)
	}
	if device.SyncState != "in_sync" || !device.Provisioned || !device.DesiredEnabled {
		return AWGDeviceInfo{}, ErrAWGOperationInProgress
	}

	err := m.deps.DB.Transaction(func(tx *gorm.DB) error {
		if findErr := tx.Where("id = ? AND client_id = ?", deviceID, clientID).First(&device).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return ErrAWGDeviceNotFound
			}
			return findErr
		}
		if device.SyncState == "pending_rotate" {
			if device.RotateRequestKey != requestKey {
				return ErrAWGOperationInProgress
			}
			return nil
		}
		if device.SyncState != "in_sync" || !device.Provisioned || !device.DesiredEnabled {
			return ErrAWGOperationInProgress
		}
		var client model.Client
		if findErr := tx.First(&client, clientID).Error; findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return ErrAWGClientInactive
			}
			return findErr
		}
		if !clientIsActiveAt(client, m.deps.Now()) {
			return ErrAWGClientInactive
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
		oldPublicKey := device.PublicKey
		newPublicKey := base64.StdEncoding.EncodeToString(keys.PublicKey)
		now := m.deps.Now()
		update := tx.Model(&model.AWGDevice{}).
			Where("id = ? AND client_id = ? AND sync_state = ? AND provisioned = ? AND desired_enabled = ? AND public_key = ?", deviceID, clientID, "in_sync", true, true, oldPublicKey).
			Updates(map[string]any{
				"rotate_request_key": requestKey, "previous_public_key": oldPublicKey,
				"crypto_context": cryptoContext, "public_key": newPublicKey,
				"private_key_enc": privateEnc, "psk_enc": pskEnc,
				"sync_state": "pending_rotate", "provisioned": false,
				"last_error": "", "rx_baseline": 0, "tx_baseline": 0,
				"last_handshake": 0, "updated_at": now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrAWGOperationInProgress
		}
		device.RotateRequestKey = requestKey
		device.PreviousPublicKey = oldPublicKey
		device.CryptoContext = cryptoContext
		device.PublicKey = newPublicKey
		device.PrivateKeyEnc = privateEnc
		device.PSKEnc = pskEnc
		device.SyncState = "pending_rotate"
		device.Provisioned = false
		device.LastError = ""
		device.RxBaseline, device.TxBaseline, device.LastHandshake = 0, 0, 0
		device.UpdatedAt = now
		return nil
	})
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	return m.resumeAWGRotation(ctx, device)
}

func (m *AWGManager) resumeAWGRotation(ctx context.Context, device model.AWGDevice) (AWGDeviceInfo, error) {
	if m.provisioner == nil {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	if err := m.provisioner.Remove(ctx, device.PreviousPublicKey); err != nil {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	var client model.Client
	if err := m.deps.DB.First(&client, device.ClientId).Error; err != nil || !clientIsActiveAt(client, m.deps.Now()) {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGClientInactive
	}
	cipher, err := m.awgCipher()
	if err != nil {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	psk, err := cipher.Decrypt(device.ClientId, device.CryptoContext, device.PSKEnc)
	if err != nil {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	defer clear(psk)
	peer := AWGPeerSpec{
		PublicKey: device.PublicKey, PresharedKey: base64.StdEncoding.EncodeToString(psk),
		AllowedIP: netipPrefix32(device.IPv4Address),
	}
	if !peer.AllowedIP.IsValid() || !peer.AllowedIP.Addr().Is4() {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	if err := m.provisioner.Add(ctx, peer); err != nil {
		m.recordAWGRotationFailure(device)
		return AWGDeviceInfo{}, ErrAWGRotationFailed
	}
	update := m.deps.DB.Model(&model.AWGDevice{}).
		Where("id = ? AND client_id = ? AND sync_state = ? AND desired_enabled = ? AND public_key = ? AND previous_public_key = ? AND rotate_request_key = ?",
			device.Id, device.ClientId, "pending_rotate", true, device.PublicKey, device.PreviousPublicKey, device.RotateRequestKey).
		Updates(map[string]any{"previous_public_key": "", "sync_state": "in_sync", "provisioned": true, "last_error": "", "updated_at": m.deps.Now()})
	if update.Error != nil {
		return AWGDeviceInfo{}, update.Error
	}
	if update.RowsAffected != 1 {
		return AWGDeviceInfo{}, errors.New("AWG rotate state changed during provisioning")
	}
	device.PreviousPublicKey, device.SyncState, device.Provisioned, device.LastError = "", "in_sync", true, ""
	return awgDeviceInfo(device), nil
}

func (m *AWGManager) recordAWGRotationFailure(device model.AWGDevice) {
	_ = m.deps.DB.Model(&model.AWGDevice{}).
		Where("id = ? AND client_id = ? AND sync_state = ? AND public_key = ? AND previous_public_key = ? AND rotate_request_key = ?",
			device.Id, device.ClientId, "pending_rotate", device.PublicKey, device.PreviousPublicKey, device.RotateRequestKey).
		Updates(map[string]any{"provisioned": false, "last_error": "AWG rotation failed", "updated_at": m.deps.Now()}).Error
}
