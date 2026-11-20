package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

var ErrAWGStatsFailed = errors.New("AWG statistics collection failed")

type AWGStatsResult struct {
	Devices  int
	Upload   uint64
	Download uint64
}

// CollectStats snapshots and accounts all managed peers in one serialized
// command and one database transaction. WireGuard receive bytes are client
// upload; transmit bytes are client download.
func (m *AWGManager) CollectStats(ctx context.Context) (AWGStatsResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return AWGStatsResult{}, err
	}
	if m == nil || m.deps.DB == nil {
		return AWGStatsResult{}, errors.New("AWG database is unavailable")
	}
	result := m.submit(ctx, func(operationCtx context.Context) awgCommandResult {
		stats, err := m.collectStatsInWorker(operationCtx)
		return awgCommandResult{value: stats, err: err}
	})
	if result.err != nil {
		return AWGStatsResult{}, result.err
	}
	stats, ok := result.value.(AWGStatsResult)
	if !ok {
		return AWGStatsResult{}, ErrAWGStatsFailed
	}
	return stats, nil
}

func (m *AWGManager) collectStatsInWorker(ctx context.Context) (AWGStatsResult, error) {
	if err := ctx.Err(); err != nil {
		return AWGStatsResult{}, err
	}
	var result AWGStatsResult
	if m.provisioner == nil {
		return result, ErrAWGProvisionerMissing
	}
	snapshot, err := m.provisioner.Snapshot(ctx)
	if err != nil {
		return result, ErrAWGStatsFailed
	}
	if len(snapshot) == 0 {
		return result, nil
	}
	keys, err := json.Marshal(awgSnapshotKeys(snapshot))
	if err != nil {
		return result, err
	}
	db := m.deps.DB.WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		var devices []struct {
			model.AWGDevice
			ClientName *string
		}
		query := tx.Model(&model.AWGDevice{}).
			Select("awg_devices.id, public_key, rx_baseline, tx_baseline, last_handshake, clients.name AS client_name").
			Joins("LEFT JOIN clients ON clients.id = awg_devices.client_id").
			Where("public_key IN (SELECT value FROM json_each(?))", string(keys))
		if m.deps.EndpointID > 0 {
			query = query.Where("endpoint_id = ?", m.deps.EndpointID)
		}
		if err := query.Scan(&devices).Error; err != nil {
			return err
		}
		clientDeltas := make(map[string]clientTrafficDelta)
		for _, device := range devices {
			if err := ctx.Err(); err != nil {
				return err
			}
			state, exists := snapshot[device.PublicKey]
			if !exists {
				continue
			}
			rxDelta := monotonicAWGDelta(state.ReceiveBytes, device.RxBaseline)
			txDelta := monotonicAWGDelta(state.TransmitBytes, device.TxBaseline)
			if rxDelta > math.MaxInt64 || txDelta > math.MaxInt64 {
				return ErrAWGStatsFailed
			}
			if device.ClientName == nil {
				return gorm.ErrRecordNotFound
			}
			name := *device.ClientName
			delta := clientDeltas[name]
			delta.up += int64(rxDelta)
			delta.down += int64(txDelta)
			clientDeltas[name] = delta
			result.Devices++
			result.Upload += rxDelta
			result.Download += txDelta
			if state.ReceiveBytes == device.RxBaseline && state.TransmitBytes == device.TxBaseline && state.LastHandshake == device.LastHandshake {
				continue
			}
			if err := tx.Model(&model.AWGDevice{}).Where("id = ?", device.Id).Updates(map[string]any{
				"rx_baseline":    state.ReceiveBytes,
				"tx_baseline":    state.TransmitBytes,
				"total_rx":       gorm.Expr("total_rx + ?", rxDelta),
				"total_tx":       gorm.Expr("total_tx + ?", txDelta),
				"last_handshake": state.LastHandshake,
				"updated_at":     m.deps.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return updateClientTrafficDeltas(tx, clientDeltas)
	})
	if err != nil {
		return AWGStatsResult{}, err
	}
	return result, nil
}

func monotonicAWGDelta(current, baseline uint64) uint64 {
	if current >= baseline {
		return current - baseline
	}
	return current
}

func awgSnapshotKeys(snapshot AWGPeerSnapshot) []string {
	keys := make([]string, 0, len(snapshot))
	for key := range snapshot {
		keys = append(keys, key)
	}
	return keys
}
