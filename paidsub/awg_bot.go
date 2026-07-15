package paidsub

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
)

const (
	awgNameStateTTL  = 5 * 60
	awgNameStateCap  = 8192
	awgOperationSize = 16
)

type awgNameKey struct {
	tgID   int64
	chatID int64
}

type awgNameState struct {
	requestKey string
	endpointID uint
	expiresAt  int64
}

// awgOps unifies the legacy single-endpoint device service and the
// endpoint-scoped manager behind the deviceID-shaped calls the bot uses.
type awgOps struct {
	list   func() ([]service.AWGDeviceInfo, error)
	get    func(deviceID uint) (service.AWGDeviceInfo, error)
	render func(ctx context.Context, deviceID uint) ([]byte, error)
	rotate func(ctx context.Context, deviceID uint, requestKey string) (service.AWGDeviceInfo, error)
	revoke func(ctx context.Context, deviceID uint) error
}

// awgDeviceEndpoint resolves the endpoint that owns deviceID for clientID.
// Zero means the device belongs to the legacy globally-configured endpoint.
// awgExpiryLine renders the device-card expiry line: empty for devices
// without a term, "Expires: in N day(s)" while active, "Expires: expired"
// after the (exclusive) boundary.
func awgExpiryLine(expiresAt, now int64, l lang) string {
	if expiresAt <= 0 {
		return ""
	}
	if expiresAt <= now {
		return tr(l, "awg_expires") + ": " + tr(l, "awg_expired")
	}
	days := (expiresAt - now + 86399) / 86400
	return tr(l, "awg_expires") + ": " + fmt.Sprintf(tr(l, "awg_expires_days"), days)
}

func awgDeviceEndpoint(clientID, deviceID uint) (uint, error) {
	var device model.AWGDevice
	if err := database.GetDB().Select("endpoint_id").Where("id = ? AND client_id = ?", deviceID, clientID).First(&device).Error; err != nil {
		return 0, err
	}
	return device.EndpointId, nil
}

func (b *Bot) awgOpsFor(clientID, deviceID uint) (awgOps, error) {
	endpointID, err := awgDeviceEndpoint(clientID, deviceID)
	if err != nil {
		return awgOps{}, err
	}
	if endpointID > 0 {
		managed := service.DefaultRuntime().AWGEndpointDeviceService()
		if managed == nil || !awgCommandsAccepted() {
			return awgOps{}, service.ErrAWGConfigUnavailable
		}
		return awgOps{
			get: func(id uint) (service.AWGDeviceInfo, error) { return managed.GetOwnedDevice(clientID, endpointID, id) },
			render: func(ctx context.Context, id uint) ([]byte, error) {
				return managed.RenderOwnedConfig(ctx, clientID, endpointID, id)
			},
			rotate: func(ctx context.Context, id uint, requestKey string) (service.AWGDeviceInfo, error) {
				return managed.RotateOwnedDevice(ctx, clientID, endpointID, id, requestKey)
			},
			revoke: func(ctx context.Context, id uint) error {
				return managed.RevokeOwnedDevice(ctx, clientID, endpointID, id)
			},
		}, nil
	}
	legacy := service.DefaultRuntime().AWGDeviceService()
	if legacy == nil || !awgCommandsAccepted() {
		return awgOps{}, service.ErrAWGConfigUnavailable
	}
	return awgOps{
		get:    func(id uint) (service.AWGDeviceInfo, error) { return legacy.GetOwnedDevice(id, clientID) },
		render: func(ctx context.Context, id uint) ([]byte, error) { return legacy.RenderOwnedConfig(ctx, id, clientID) },
		rotate: func(ctx context.Context, id uint, requestKey string) (service.AWGDeviceInfo, error) {
			return legacy.RotateOwnedDevice(ctx, id, clientID, requestKey)
		},
		revoke: func(ctx context.Context, id uint) error { return legacy.RevokeOwnedDevice(ctx, id, clientID) },
	}, nil
}

func (b *Bot) awgClient(tgID int64) (uint, service.AWGDeviceService, error) {
	client, err := b.svc.ClientByTgUserId(tgID)
	if err != nil {
		return 0, nil, err
	}
	devices := service.DefaultRuntime().AWGDeviceService()
	if devices == nil || !awgCommandsAccepted() {
		return 0, nil, service.ErrAWGConfigUnavailable
	}
	return client.Id, devices, nil
}

func (b *Bot) allowAWG(tgID int64) bool {
	return b.awgLimiter != nil && b.awgLimiter.allow(tgID, nowUnix())
}

func (b *Bot) cmdAWGList(ctx context.Context, chatID, tgID int64, l lang) {
	if !b.allowAWG(tgID) {
		_ = b.sendMessage(ctx, chatID, tr(l, "rate_limited"), nil)
		return
	}
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
		return
	}
	rows := make([][]inlineButton, 0, 8)
	accesses, accessErr := service.ListClientAWGEndpointAccess(database.GetDB(), clientID)
	if accessErr == nil && len(accesses) > 0 {
		managed := service.DefaultRuntime().AWGEndpointDeviceService()
		if managed == nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
			return
		}
		for _, access := range accesses {
			list, listErr := managed.ListDevices(clientID, access.EndpointID)
			if listErr != nil {
				continue
			}
			for _, device := range list {
				if !device.DesiredEnabled {
					continue
				}
				label := device.Name
				if len(accesses) > 1 {
					label = access.Tag + " · " + label
				}
				if !device.Provisioned {
					label += " (pending)"
				}
				rows = append(rows, []inlineButton{{Text: label, CallbackData: fmt.Sprintf("awg:v:%d", device.ID)}})
			}
			addLabel := tr(l, "awg_add")
			if len(accesses) > 1 {
				addLabel += " · " + access.Tag
			}
			rows = append(rows, []inlineButton{{Text: addLabel, CallbackData: fmt.Sprintf("awg:add:%d", access.EndpointID)}})
		}
	} else {
		list, listErr := devices.ListDevices(clientID)
		if listErr != nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
			return
		}
		for _, device := range list {
			label := device.Name
			if !device.Provisioned {
				label += " (pending)"
			}
			rows = append(rows, []inlineButton{{Text: label, CallbackData: fmt.Sprintf("awg:v:%d", device.ID)}})
		}
		rows = append(rows, []inlineButton{{Text: tr(l, "awg_add"), CallbackData: "awg:add"}})
	}
	rows = append(rows, []inlineButton{{Text: tr(l, "menu_back"), CallbackData: "menu"}})
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_title"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) cmdAWGCreate(ctx context.Context, chatID, tgID int64, endpointID uint, callbackID string, l lang) {
	if !b.allowAWG(tgID) {
		_ = b.sendMessage(ctx, chatID, tr(l, "rate_limited"), nil)
		return
	}
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
		return
	}
	if endpointID > 0 {
		if _, _, _, accessErr := service.EffectiveAWGEndpointAccess(database.GetDB(), clientID, endpointID); accessErr != nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
			return
		}
	}
	requestKey, ok := awgOperationKey("tg-create", tgID, endpointID, callbackID)
	if !ok || !b.beginAWGName(tgID, chatID, requestKey, endpointID, nowUnix()) {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	rows := [][]inlineButton{{{Text: tr(l, "menu_back"), CallbackData: "awg:list"}}}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_name_prompt"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) createNamedAWG(ctx context.Context, chatID, tgID int64, name string, state awgNameState, l lang) {
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
		return
	}
	var device service.AWGDeviceInfo
	if state.endpointID > 0 {
		managed := service.DefaultRuntime().AWGEndpointDeviceService()
		if managed == nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
			return
		}
		// Bot-created devices have no expiry (0 = never); setting a term from
		// the bot is a possible follow-up, not part of this stage.
		device, err = managed.CreateDevice(ctx, clientID, state.endpointID, state.requestKey, name, 0)
	} else {
		settings, settingsErr := (&service.SettingService{}).GetAWGSettings()
		if settingsErr != nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
			return
		}
		limit, limitErr := (&TariffService{}).EffectiveAWGDeviceLimit(clientID, settings.DefaultDeviceLimit)
		if limitErr != nil {
			_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
			return
		}
		device, err = devices.CreateDevice(ctx, clientID, state.requestKey, name, limit, 0)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAWGInvalidDeviceName):
			b.restoreAWGName(tgID, chatID, state, nowUnix())
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_name_invalid"), nil)
		case errors.Is(err, service.ErrAWGDeviceLimitReached):
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_limit"), nil)
		default:
			_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		}
		return
	}
	b.cmdAWGView(ctx, chatID, tgID, device.ID, l)
}

func (b *Bot) cmdAWGView(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	device, err := ops.get(deviceID)
	if err != nil {
		return
	}
	text := fmt.Sprintf("%s\nIP: %s\nTraffic: %s / %s\nHandshake: %d\nState: %s",
		device.Name, device.IPv4Address, humanBytesU64(device.TotalRx), humanBytesU64(device.TotalTx), device.LastHandshake, device.SyncState)
	if line := awgExpiryLine(device.ExpiresAt, nowUnix(), l); line != "" {
		text += "\n" + line
	}
	rows := [][]inlineButton{}
	if device.Provisioned {
		rows = append(rows, []inlineButton{{Text: tr(l, "awg_config"), CallbackData: fmt.Sprintf("awg:c:%d", device.ID)}, {Text: "QR", CallbackData: fmt.Sprintf("awg:q:%d", device.ID)}})
	}
	rows = append(rows, []inlineButton{{Text: tr(l, "awg_rotate"), CallbackData: fmt.Sprintf("awg:r:%d", device.ID)}, {Text: tr(l, "awg_delete"), CallbackData: fmt.Sprintf("awg:d:%d", device.ID)}})
	rows = append(rows, []inlineButton{{Text: tr(l, "menu_back"), CallbackData: "awg:list"}})
	_ = b.sendMessage(ctx, chatID, text, &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) cmdAWGConfig(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	if !b.allowAWG(tgID) {
		return
	}
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	config, err := ops.render(ctx, deviceID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	if err := b.sendAWGDocument(ctx, chatID, deviceID, config); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
	}
}

func (b *Bot) cmdAWGQR(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	if !b.allowAWG(tgID) {
		return
	}
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	config, err := ops.render(ctx, deviceID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	png, err := service.RenderAWGConfigQR(config)
	if errors.Is(err, service.ErrAWGConfigTooLargeQR) {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_qr_large"), nil)
		return
	}
	if err != nil || b.sendPhoto(ctx, chatID, png, tr(l, "awg_qr")) != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
	}
}

func (b *Bot) confirmAWGRotate(ctx context.Context, chatID, tgID int64, deviceID uint, callbackID string, l lang) {
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	if _, err := ops.get(deviceID); err != nil {
		return
	}
	nonce, ok := awgOperationNonce(callbackID)
	if !ok {
		return
	}
	rows := [][]inlineButton{{{Text: tr(l, "awg_confirm"), CallbackData: fmt.Sprintf("awg:ry:%d:%s", deviceID, nonce)}}, {{Text: tr(l, "menu_back"), CallbackData: fmt.Sprintf("awg:v:%d", deviceID)}}}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_rotate_confirm"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) rotateAWG(ctx context.Context, chatID, tgID int64, deviceID uint, nonce string, l lang) {
	if !b.allowAWG(tgID) {
		return
	}
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	requestKey := fmt.Sprintf("tg-rotate:%d:%d:%s", tgID, deviceID, nonce)
	if _, err := ops.rotate(ctx, deviceID, requestKey); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	b.cmdAWGView(ctx, chatID, tgID, deviceID, l)
}

func (b *Bot) confirmAWGDelete(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	if _, err := ops.get(deviceID); err != nil {
		return
	}
	rows := [][]inlineButton{{{Text: tr(l, "awg_confirm"), CallbackData: fmt.Sprintf("awg:dy:%d", deviceID)}}, {{Text: tr(l, "menu_back"), CallbackData: fmt.Sprintf("awg:v:%d", deviceID)}}}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_delete_confirm"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) deleteAWG(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	if !b.allowAWG(tgID) {
		return
	}
	clientID, _, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	ops, err := b.awgOpsFor(clientID, deviceID)
	if err != nil {
		return
	}
	if err := ops.revoke(ctx, deviceID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_deleted"), &inlineKeyboard{InlineKeyboard: [][]inlineButton{{{Text: tr(l, "menu_back"), CallbackData: "awg:list"}}}})
}

func awgCallbackTooLong(data string) bool { return len(strings.TrimSpace(data)) > 64 }

func (b *Bot) beginAWGName(tgID, chatID int64, requestKey string, endpointID uint, now int64) bool {
	b.awgNameMu.Lock()
	defer b.awgNameMu.Unlock()
	if b.awgNames == nil {
		b.awgNames = make(map[awgNameKey]awgNameState)
	}
	b.gcAWGNamesLocked(now)
	key := awgNameKey{tgID: tgID, chatID: chatID}
	if _, exists := b.awgNames[key]; !exists && len(b.awgNames) >= awgNameStateCap {
		return false
	}
	b.awgNames[key] = awgNameState{requestKey: requestKey, endpointID: endpointID, expiresAt: now + awgNameStateTTL}
	return true
}

func (b *Bot) takeAWGNameState(tgID, chatID, now int64) (awgNameState, bool) {
	b.awgNameMu.Lock()
	defer b.awgNameMu.Unlock()
	key := awgNameKey{tgID: tgID, chatID: chatID}
	state, ok := b.awgNames[key]
	if !ok || state.expiresAt <= now {
		delete(b.awgNames, key)
		return awgNameState{}, false
	}
	delete(b.awgNames, key)
	return state, true
}

func (b *Bot) restoreAWGName(tgID, chatID int64, state awgNameState, now int64) {
	if state.expiresAt <= now {
		return
	}
	b.awgNameMu.Lock()
	if b.awgNames == nil {
		b.awgNames = make(map[awgNameKey]awgNameState)
	}
	b.awgNames[awgNameKey{tgID: tgID, chatID: chatID}] = state
	b.awgNameMu.Unlock()
}

func (b *Bot) cancelAWGName(tgID, chatID int64) {
	b.awgNameMu.Lock()
	delete(b.awgNames, awgNameKey{tgID: tgID, chatID: chatID})
	b.awgNameMu.Unlock()
}

func (b *Bot) clearAWGNameStates() {
	b.awgNameMu.Lock()
	b.awgNames = make(map[awgNameKey]awgNameState)
	b.awgNameMu.Unlock()
}

func (b *Bot) gcAWGNamesLocked(now int64) {
	for key, state := range b.awgNames {
		if state.expiresAt <= now {
			delete(b.awgNames, key)
		}
	}
}

func awgOperationNonce(callbackID string) (string, bool) {
	callbackID = strings.TrimSpace(callbackID)
	if callbackID == "" {
		return "", false
	}
	sum := sha256.Sum256([]byte(callbackID))
	nonce := fmt.Sprintf("%x", sum[:awgOperationSize/2])
	return nonce, true
}

func awgOperationKey(prefix string, tgID int64, deviceID uint, callbackID string) (string, bool) {
	nonce, ok := awgOperationNonce(callbackID)
	if !ok {
		return "", false
	}
	if deviceID == 0 {
		return fmt.Sprintf("%s:%d:%s", prefix, tgID, nonce), true
	}
	return fmt.Sprintf("%s:%d:%d:%s", prefix, tgID, deviceID, nonce), true
}

func parseAWGConfirmation(data, prefix string) (uint, string, bool) {
	parts := strings.Split(strings.TrimPrefix(data, prefix), ":")
	if len(parts) != 2 || len(parts[1]) != awgOperationSize {
		return 0, "", false
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		return 0, "", false
	}
	for _, r := range parts[1] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return 0, "", false
		}
	}
	return uint(id), parts[1], true
}
