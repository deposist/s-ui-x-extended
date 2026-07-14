package paidsub

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	expiresAt  int64
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
	list, err := devices.ListDevices(clientID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	rows := make([][]inlineButton, 0, len(list)+2)
	for _, device := range list {
		label := device.Name
		if !device.Provisioned {
			label += " (pending)"
		}
		rows = append(rows, []inlineButton{{Text: label, CallbackData: fmt.Sprintf("awg:v:%d", device.ID)}})
	}
	rows = append(rows, []inlineButton{{Text: tr(l, "awg_add"), CallbackData: "awg:add"}})
	rows = append(rows, []inlineButton{{Text: tr(l, "menu_back"), CallbackData: "menu"}})
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_title"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) cmdAWGCreate(ctx context.Context, chatID, tgID int64, callbackID string, l lang) {
	if !b.allowAWG(tgID) {
		_ = b.sendMessage(ctx, chatID, tr(l, "rate_limited"), nil)
		return
	}
	if _, _, err := b.awgClient(tgID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
		return
	}
	requestKey, ok := awgOperationKey("tg-create", tgID, 0, callbackID)
	if !ok || !b.beginAWGName(tgID, chatID, requestKey, nowUnix()) {
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
	settings, err := (&service.SettingService{}).GetAWGSettings()
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_unavailable"), nil)
		return
	}
	limit, err := (&TariffService{}).EffectiveAWGDeviceLimit(clientID, settings.DefaultDeviceLimit)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	device, err := devices.CreateDevice(ctx, clientID, state.requestKey, name, limit)
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
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	device, err := devices.GetOwnedDevice(deviceID, clientID)
	if err != nil {
		return
	}
	text := fmt.Sprintf("%s\nIP: %s\nTraffic: %s / %s\nHandshake: %d\nState: %s",
		device.Name, device.IPv4Address, humanBytesU64(device.TotalRx), humanBytesU64(device.TotalTx), device.LastHandshake, device.SyncState)
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
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	config, err := devices.RenderOwnedConfig(ctx, deviceID, clientID)
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
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	config, err := devices.RenderOwnedConfig(ctx, deviceID, clientID)
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
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	if _, err := devices.GetOwnedDevice(deviceID, clientID); err != nil {
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
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	requestKey := fmt.Sprintf("tg-rotate:%d:%d:%s", tgID, deviceID, nonce)
	if _, err := devices.RotateOwnedDevice(ctx, deviceID, clientID, requestKey); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	b.cmdAWGView(ctx, chatID, tgID, deviceID, l)
}

func (b *Bot) confirmAWGDelete(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	if _, err := devices.GetOwnedDevice(deviceID, clientID); err != nil {
		return
	}
	rows := [][]inlineButton{{{Text: tr(l, "awg_confirm"), CallbackData: fmt.Sprintf("awg:dy:%d", deviceID)}}, {{Text: tr(l, "menu_back"), CallbackData: fmt.Sprintf("awg:v:%d", deviceID)}}}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_delete_confirm"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) deleteAWG(ctx context.Context, chatID, tgID int64, deviceID uint, l lang) {
	if !b.allowAWG(tgID) {
		return
	}
	clientID, devices, err := b.awgClient(tgID)
	if err != nil {
		return
	}
	if err := devices.RevokeOwnedDevice(ctx, deviceID, clientID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "awg_pending"), nil)
		return
	}
	_ = b.sendMessage(ctx, chatID, tr(l, "awg_deleted"), &inlineKeyboard{InlineKeyboard: [][]inlineButton{{{Text: tr(l, "menu_back"), CallbackData: "awg:list"}}}})
}

func awgCallbackTooLong(data string) bool { return len(strings.TrimSpace(data)) > 64 }

func (b *Bot) beginAWGName(tgID, chatID int64, requestKey string, now int64) bool {
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
	b.awgNames[key] = awgNameState{requestKey: requestKey, expiresAt: now + awgNameStateTTL}
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
