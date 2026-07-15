package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"

	"github.com/deposist/s-ui-x-extended/database/model"
)

var ErrAWGReconcileFailed = errors.New("AWG reconcile failed")

// AWGReconcileResult reports drift repaired during one serialized pass.
type AWGReconcileResult struct {
	Added   int
	Removed int
	Desired int
}

// Reconcile repairs live and persisted peer state from the durable device and
// client tables. It runs as one manager command, so no other UAPI operation can
// interleave with the snapshot and repairs.
func (m *AWGManager) Reconcile(ctx context.Context) (AWGReconcileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AWGReconcileResult{}, err
	}
	if m == nil || m.deps.DB == nil {
		return AWGReconcileResult{}, errors.New("AWG database is unavailable")
	}
	result := m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		reconcileResult, err := m.reconcileInWorker(operationCtx)
		return awgCommandResult{value: reconcileResult, err: err}
	})
	if result.err != nil {
		return AWGReconcileResult{}, result.err
	}
	reconcileResult, ok := result.value.(AWGReconcileResult)
	if !ok {
		return AWGReconcileResult{}, ErrAWGReconcileFailed
	}
	return reconcileResult, nil
}

type awgDesiredPeer struct {
	device model.AWGDevice
	peer   AWGPeerSpec
}

func (m *AWGManager) reconcileInWorker(ctx context.Context) (AWGReconcileResult, error) {
	var result AWGReconcileResult
	if m.provisioner == nil {
		return result, ErrAWGProvisionerMissing
	}
	settings, err := m.deps.LoadSettings()
	if err != nil {
		return result, ErrAWGReconcileFailed
	}
	if !settings.Enabled {
		return result, nil
	}
	if _, err := m.deps.LoadEndpoint(m.deps.DB, settings); err != nil {
		return result, ErrAWGReconcileFailed
	}

	var devices []model.AWGDevice
	deviceQuery := m.deps.DB.Order("id")
	if m.deps.EndpointID > 0 {
		deviceQuery = deviceQuery.Where("endpoint_id = ?", m.deps.EndpointID)
	}
	if err := deviceQuery.Find(&devices).Error; err != nil {
		return result, err
	}
	clientIDs := make([]uint, 0, len(devices))
	seenClient := make(map[uint]struct{}, len(devices))
	for _, device := range devices {
		if _, exists := seenClient[device.ClientId]; !exists {
			seenClient[device.ClientId] = struct{}{}
			clientIDs = append(clientIDs, device.ClientId)
		}
	}
	var clients []model.Client
	if len(clientIDs) > 0 {
		if err := m.deps.DB.Where("id IN ?", clientIDs).Find(&clients).Error; err != nil {
			return result, err
		}
	}
	clientsByID := make(map[uint]model.Client, len(clients))
	for _, client := range clients {
		clientsByID[client.Id] = client
	}

	desired := make(map[string]awgDesiredPeer, len(devices))
	invalid := make(map[uint]struct{})
	for _, device := range devices {
		client, clientExists := clientsByID[device.ClientId]
		if !device.DesiredEnabled || !clientExists || !clientIsActiveAt(client, m.deps.Now()) || deviceExpiredAt(device, m.deps.Now()) {
			continue
		}
		peer, peerErr := m.awgPeerSpec(device)
		if peerErr != nil {
			m.recordAWGReconcileFailure(device.Id)
			invalid[device.Id] = struct{}{}
			continue
		}
		desired[device.PublicKey] = awgDesiredPeer{device: device, peer: peer}
	}
	result.Desired = len(desired)

	snapshot, err := m.provisioner.Snapshot(ctx)
	if err != nil {
		return result, ErrAWGReconcileFailed
	}

	removeKeys := make([]string, 0)
	for publicKey := range snapshot {
		if _, keep := desired[publicKey]; !keep {
			removeKeys = append(removeKeys, publicKey)
		}
	}
	for _, device := range devices {
		if device.PreviousPublicKey != "" {
			removeKeys = append(removeKeys, device.PreviousPublicKey)
		}
	}
	sort.Strings(removeKeys)
	removeKeys = uniqueAWGStrings(removeKeys)
	for _, publicKey := range removeKeys {
		if err := m.provisioner.Remove(ctx, publicKey); err != nil {
			return result, ErrAWGReconcileFailed
		}
		result.Removed++
	}

	publicKeys := make([]string, 0, len(desired))
	for publicKey := range desired {
		publicKeys = append(publicKeys, publicKey)
	}
	sort.Strings(publicKeys)
	for _, publicKey := range publicKeys {
		wanted := desired[publicKey]
		state, exists := snapshot[publicKey]
		if exists && len(state.AllowedIPs) == 1 && state.AllowedIPs[0] == wanted.peer.AllowedIP.Masked() {
			continue
		}
		if err := m.provisioner.Add(ctx, wanted.peer); err != nil {
			m.recordAWGReconcileFailure(wanted.device.Id)
			return result, ErrAWGReconcileFailed
		}
		result.Added++
	}

	now := m.deps.Now()
	for _, device := range devices {
		if _, failed := invalid[device.Id]; failed {
			continue
		}
		_, enabled := desired[device.PublicKey]
		updates := map[string]any{"updated_at": now, "last_error": ""}
		if enabled {
			updates["sync_state"] = "in_sync"
			updates["provisioned"] = true
			updates["previous_public_key"] = ""
		} else {
			updates["sync_state"] = "in_sync"
			updates["provisioned"] = false
			updates["previous_public_key"] = ""
			if !device.DesiredEnabled && device.RevokedAt == 0 {
				updates["revoked_at"] = now
				updates["ip_reusable_after"] = now + awgRevokedIPQuarantineSeconds
			}
		}
		if err := m.deps.DB.Model(&model.AWGDevice{}).Where("id = ?", device.Id).Updates(updates).Error; err != nil {
			return result, err
		}
	}

	if m.deps.SyncEndpointPeers != nil {
		persisted := make([]AWGPersistedPeer, 0, len(publicKeys))
		for _, publicKey := range publicKeys {
			peer := desired[publicKey].peer
			persisted = append(persisted, AWGPersistedPeer{
				PublicKey:  peer.PublicKey,
				AllowedIPs: []string{peer.AllowedIP.String()},
			})
		}
		if err := m.deps.SyncEndpointPeers(m.deps.DB, settings, persisted); err != nil {
			return result, ErrAWGReconcileFailed
		}
	}
	return result, nil
}

func (m *AWGManager) awgPeerSpec(device model.AWGDevice) (AWGPeerSpec, error) {
	cipher, err := m.awgCipher()
	if err != nil {
		return AWGPeerSpec{}, err
	}
	psk, err := cipher.Decrypt(device.ClientId, device.CryptoContext, device.PSKEnc)
	if err != nil {
		return AWGPeerSpec{}, err
	}
	defer clear(psk)
	if len(psk) != 32 {
		return AWGPeerSpec{}, fmt.Errorf("invalid AWG key material")
	}
	allowedIP := netipPrefix32(device.IPv4Address)
	if !allowedIP.IsValid() || !allowedIP.Addr().Is4() {
		return AWGPeerSpec{}, fmt.Errorf("invalid AWG device address")
	}
	return AWGPeerSpec{
		PublicKey: device.PublicKey, PresharedKey: base64.StdEncoding.EncodeToString(psk), AllowedIP: allowedIP,
	}, nil
}

func (m *AWGManager) recordAWGReconcileFailure(deviceID uint) {
	_ = m.deps.DB.Model(&model.AWGDevice{}).Where("id = ?", deviceID).
		Updates(map[string]any{"provisioned": false, "last_error": "AWG reconciliation failed", "updated_at": m.deps.Now()}).Error
}

func uniqueAWGStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
