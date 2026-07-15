package service

import (
	"context"
	"errors"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

const awgRevokedIPQuarantineSeconds int64 = 24 * 60 * 60

var (
	ErrAWGDeviceNotFound   = errors.New("AWG device not found")
	ErrAWGRevocationFailed = errors.New("AWG revocation failed")
)

// ListDevices returns only the non-secret device projection owned by clientID.
func (m *AWGManager) ListDevices(clientID uint) ([]AWGDeviceInfo, error) {
	if m == nil || m.deps.DB == nil {
		return nil, errors.New("AWG database is unavailable")
	}
	var devices []model.AWGDevice
	if err := m.deps.DB.Where("client_id = ? AND endpoint_id = ?", clientID, m.deps.EndpointID).Order("created_at, id").Find(&devices).Error; err != nil {
		return nil, err
	}
	result := make([]AWGDeviceInfo, len(devices))
	for i := range devices {
		result[i] = awgDeviceInfo(devices[i])
	}
	return result, nil
}

// GetOwnedDevice deliberately maps absent and foreign IDs to the same error.
func (m *AWGManager) GetOwnedDevice(deviceID, clientID uint) (AWGDeviceInfo, error) {
	if m == nil || m.deps.DB == nil {
		return AWGDeviceInfo{}, errors.New("AWG database is unavailable")
	}
	var device model.AWGDevice
	err := m.deps.DB.Where("id = ? AND client_id = ? AND endpoint_id = ?", deviceID, clientID, m.deps.EndpointID).First(&device).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AWGDeviceInfo{}, ErrAWGDeviceNotFound
	}
	if err != nil {
		return AWGDeviceInfo{}, err
	}
	return awgDeviceInfo(device), nil
}

// RevokeOwnedDevice serializes the durable intent and every live peer removal
// in one worker command. A failed live removal remains pending for reconciliation.
func (m *AWGManager) RevokeOwnedDevice(ctx context.Context, deviceID, clientID uint) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil || m.deps.DB == nil {
		return errors.New("AWG database is unavailable")
	}
	return m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		return awgCommandResult{err: m.revokeOwnedDeviceInWorker(operationCtx, deviceID, clientID)}
	}).err
}

func (m *AWGManager) revokeOwnedDeviceInWorker(ctx context.Context, deviceID, clientID uint) error {
	now := m.deps.Now()
	var device model.AWGDevice
	alreadyRevoked := false
	err := m.deps.DB.Transaction(func(tx *gorm.DB) error {
		findErr := tx.Where("id = ? AND client_id = ? AND endpoint_id = ?", deviceID, clientID, m.deps.EndpointID).First(&device).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return ErrAWGDeviceNotFound
		}
		if findErr != nil {
			return findErr
		}
		if !device.DesiredEnabled && device.SyncState == "in_sync" && !device.Provisioned && device.RevokedAt != 0 {
			alreadyRevoked = true
			return nil
		}
		return tx.Model(&model.AWGDevice{}).
			Where("id = ? AND client_id = ? AND endpoint_id = ?", deviceID, clientID, m.deps.EndpointID).
			Updates(map[string]any{
				"desired_enabled": false,
				"sync_state":      "pending_remove",
				"last_error":      "",
				"updated_at":      now,
			}).Error
	})
	if err != nil || alreadyRevoked {
		return err
	}

	keys := make([]string, 0, 2)
	if device.PreviousPublicKey != "" {
		keys = append(keys, device.PreviousPublicKey)
	}
	if device.PublicKey != "" && device.PublicKey != device.PreviousPublicKey {
		keys = append(keys, device.PublicKey)
	}
	for _, publicKey := range keys {
		if m.provisioner == nil || m.provisioner.Remove(ctx, publicKey) != nil {
			_ = m.deps.DB.Model(&model.AWGDevice{}).
				Where("id = ? AND client_id = ? AND desired_enabled = ? AND sync_state = ?", deviceID, clientID, false, "pending_remove").
				Updates(map[string]any{"last_error": "AWG revocation failed", "updated_at": m.deps.Now()}).Error
			return ErrAWGRevocationFailed
		}
	}

	finalNow := m.deps.Now()
	result := m.deps.DB.Model(&model.AWGDevice{}).
		Where("id = ? AND client_id = ? AND desired_enabled = ? AND sync_state = ?", deviceID, clientID, false, "pending_remove").
		Updates(map[string]any{
			"sync_state":          "in_sync",
			"provisioned":         false,
			"previous_public_key": "",
			"last_error":          "",
			"revoked_at":          finalNow,
			"ip_reusable_after":   finalNow + awgRevokedIPQuarantineSeconds,
			"updated_at":          finalNow,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("AWG revoke state changed during provisioning")
	}
	return nil
}
