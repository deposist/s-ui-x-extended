package paidsub

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

// tryAutoRegister creates-and-binds a new trial client for an unknown Telegram
// user when auto-registration is enabled. Returns true if it handled the
// interaction. All client data is built server-side; nothing from the message
// except the (validated) numeric id and a sanitized username is used.
func (b *Bot) tryAutoRegister(ctx context.Context, chatID int64, from *tgUser, l lang) bool {
	enabled, err := b.setting.GetPaidSubAutoRegister()
	if err != nil || !enabled {
		return false
	}

	// Per-user /start rate limit (anti-DoS for registration).
	maxStart, _ := b.setting.GetPaidSubStartRateLimitPerMin()
	if !b.startLimiter.allowWithMax(from.ID, nowUnix(), maxStart) {
		_ = b.sendMessage(ctx, chatID, tr(l, "rate_limited"), nil)
		return true
	}

	inbounds, _ := b.setting.GetPaidSubAutoInbounds()
	if len(inbounds) == 0 {
		_ = b.sendMessage(ctx, chatID, tr(l, "reg_no_setup"), nil)
		return true
	}

	db := database.GetDB()

	// Global cap on auto-registered clients (anti-DoS).
	if maxClients, _ := b.setting.GetPaidSubMaxClients(); maxClients > 0 {
		var cnt int64
		if err := db.Model(&Binding{}).Count(&cnt).Error; err != nil {
			// Fail closed: if the anti-DoS cap cannot be evaluated, refuse rather
			// than risk unbounded auto-registration.
			logger.Warning("paidsub: count bindings for cap failed: ", err)
			_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
			return true
		}
		if cnt >= int64(maxClients) {
			_ = b.sendMessage(ctx, chatID, tr(l, "reg_full"), nil)
			return true
		}
	}

	name := b.uniqueClientName(db, from.ID)
	config, err := generateClientConfig(name)
	if err != nil {
		logger.Warning("paidsub: generate client config: ", err)
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return true
	}

	trialDays, _ := b.setting.GetPaidSubTrialDays()
	trialGB, _ := b.setting.GetPaidSubTrialVolumeGB()
	var expiry int64
	if trialDays > 0 {
		expiry = nowUnix() + int64(trialDays)*86400
	}
	var volume int64
	if trialGB > 0 {
		volume = int64(trialGB) * 1024 * 1024 * 1024
	}

	inboundsJSON, _ := json.Marshal(inbounds)
	payload := map[string]any{
		"enable":   true,
		"name":     name,
		"config":   json.RawMessage(config),
		"inbounds": json.RawMessage(inboundsJSON),
		"volume":   volume,
		"expiry":   expiry,
		"desc":     sanitizeUsername(from.Username),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return true
	}

	host, _ := b.setting.GetWebDomain()
	cfg := service.NewConfigServiceWithRuntime(service.DefaultRuntime())
	if _, err := cfg.Save("clients", "new", data, "", "PaidSubBot", host); err != nil {
		logger.Warning("paidsub: auto-register save failed: ", err)
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return true
	}

	var created model.Client
	if err := db.Where("name = ?", name).First(&created).Error; err != nil {
		logger.Warning("paidsub: auto-register lookup failed: ", err)
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return true
	}
	if err := b.svc.SetBinding(created.Id, from.ID); err != nil {
		logger.Warning("paidsub: auto-register bind failed: ", err)
	}
	_ = (&service.AuditService{}).Record(service.AuditEvent{
		Actor:    "PaidSubBot",
		Event:    "paidsub_registered",
		Resource: "paidsub",
		Severity: service.AuditSeverityInfo,
		Details:  map[string]any{"clientId": created.Id, "tgUserId": from.ID},
	})
	_ = b.sendMessage(ctx, chatID, tr(l, "registered"), b.menuKeyboard(l))
	return true
}

// uniqueClientName derives a collision-free name from the Telegram id.
func (b *Bot) uniqueClientName(db *gorm.DB, tgID int64) string {
	base := "tg" + strconv.FormatInt(tgID, 10)
	name := base
	for i := 0; i < 50; i++ {
		var cnt int64
		db.Model(&model.Client{}).Where("name = ?", name).Count(&cnt)
		if cnt == 0 {
			return name
		}
		name = base + "_" + common.Random(4)
	}
	return base + "_" + common.Random(8)
}

// generateClientConfig builds a full per-protocol client config, mirroring the
// panel's randomConfigs (frontend/src/types/clients.ts) so the resulting client
// works on any assigned inbound type. Both shadowsocks key lengths are emitted
// (32-byte for legacy/2022-256, 16-byte for 2022-blake3-aes-128-gcm).
func generateClientConfig(name string) (json.RawMessage, error) {
	mixedPassword := common.Random(10)
	ss32, err := randomSSPassword(32)
	if err != nil {
		return nil, err
	}
	ss16, err := randomSSPassword(16)
	if err != nil {
		return nil, err
	}
	u, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	uuidStr := u.String()
	cfg := map[string]map[string]any{
		"mixed":         {"username": name, "password": mixedPassword},
		"socks":         {"username": name, "password": mixedPassword},
		"http":          {"username": name, "password": mixedPassword},
		"shadowsocks":   {"name": name, "password": ss32},
		"shadowsocks16": {"name": name, "password": ss16},
		"shadowtls":     {"name": name, "password": ss32},
		"vmess":         {"name": name, "uuid": uuidStr, "alterId": 0},
		"vless":         {"name": name, "uuid": uuidStr, "flow": "xtls-rprx-vision"},
		"anytls":        {"name": name, "password": mixedPassword},
		"trojan":        {"name": name, "password": mixedPassword},
		"naive":         {"username": name, "password": mixedPassword},
		"hysteria":      {"name": name, "auth_str": mixedPassword},
		"tuic":          {"name": name, "uuid": uuidStr, "password": mixedPassword},
		"hysteria2":     {"name": name, "password": mixedPassword},
	}
	return json.Marshal(cfg)
}

func randomSSPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// sanitizeUsername keeps only safe characters for display in the client's desc.
// It is never used in SQL (parameterized queries everywhere).
func sanitizeUsername(username string) string {
	if username == "" {
		return "telegram"
	}
	var sb strings.Builder
	sb.WriteByte('@')
	for _, r := range username {
		if r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			sb.WriteRune(r)
		}
	}
	out := sb.String()
	if out == "@" {
		return "telegram"
	}
	return out
}
