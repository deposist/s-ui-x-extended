package ipmonitor

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/util/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ModeMonitor = "monitor"
	ModeEnforce = "enforce"

	allowCacheTTL          = 30 * time.Second
	securityEventDebounce  = 60 * time.Second
	securityEventMaxMapAge = time.Hour
	ipMaskPrefix           = 12
)

type pendingIP struct {
	lastSeen int64
	display  *string
}

var pending = struct {
	sync.Mutex
	byClient map[string]map[string]pendingIP
}{
	byClient: map[string]map[string]pendingIP{},
}

type allowCacheEntry struct {
	limit     int
	mode      string
	ips       map[string]struct{}
	expiresAt time.Time
}

var allowCache = struct {
	sync.Mutex
	byClient map[string]allowCacheEntry
}{
	byClient: map[string]allowCacheEntry{},
}

var allowCacheRefresh = struct {
	sync.Mutex
	inFlight map[string]struct{}
}{
	inFlight: map[string]struct{}{},
}

var securityEvents = struct {
	sync.Mutex
	lastEmittedAt map[string]time.Time
}{
	lastEmittedAt: map[string]time.Time{},
}

var ipHashSalt = struct {
	sync.Mutex
	value []byte
}{}

var ipPrivacySettings = struct {
	sync.Mutex
	showRaw   bool
	expiresAt time.Time
}{}

func init() {
	database.RegisterResetHook("ipmonitor", ResetCaches)
}

func ResetCaches() {
	pending.Lock()
	pending.byClient = map[string]map[string]pendingIP{}
	pending.Unlock()

	allowCache.Lock()
	allowCache.byClient = map[string]allowCacheEntry{}
	allowCache.Unlock()

	allowCacheRefresh.Lock()
	allowCacheRefresh.inFlight = map[string]struct{}{}
	allowCacheRefresh.Unlock()

	securityEvents.Lock()
	securityEvents.lastEmittedAt = map[string]time.Time{}
	securityEvents.Unlock()

	ipHashSalt.Lock()
	ipHashSalt.value = nil
	ipHashSalt.Unlock()

	ipPrivacySettings.Lock()
	ipPrivacySettings.showRaw = false
	ipPrivacySettings.expiresAt = time.Time{}
	ipPrivacySettings.Unlock()
}

func Record(clientName string, ip string) {
	if clientName == "" || ip == "" {
		return
	}
	ipHash, display, ok := recordIPFields(ip)
	if !ok {
		return
	}
	now := time.Now().Unix()
	pending.Lock()
	if pending.byClient[clientName] == nil {
		pending.byClient[clientName] = map[string]pendingIP{}
	}
	pending.byClient[clientName][ipHash] = pendingIP{
		lastSeen: now,
		display:  display,
	}
	pending.Unlock()
	cacheAddIP(clientName, ipHash)
}

func Allow(clientName string, ip string) bool {
	if clientName == "" || ip == "" {
		return true
	}
	ipHash, err := hashIP(ip)
	if err != nil {
		return true
	}
	entry, ok := cachedClient(clientName, time.Now())
	if !ok {
		refreshClientAsync(clientName)
		return true
	}
	if entry.mode != ModeEnforce || entry.limit <= 0 {
		return true
	}
	seen := map[string]struct{}{ipHash: {}}
	for seenHash := range entry.ips {
		seen[seenHash] = struct{}{}
	}
	pending.Lock()
	for seenHash := range pending.byClient[clientName] {
		seen[seenHash] = struct{}{}
	}
	pending.Unlock()
	if len(seen) <= entry.limit {
		return true
	}
	publishSecurityEvent(clientName, "ip_enforced_reject", map[string]any{
		"kind":   "ip_enforced_reject",
		"client": clientName,
		"ipHash": ipHash,
		"limit":  entry.limit,
		"count":  len(seen),
	})
	return false
}

func WarmUp() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	if _, err := getInstallSalt(); err != nil {
		return err
	}
	entries, err := loadActiveEnforceEntries(db, time.Now())
	if err != nil {
		return err
	}
	allowCache.Lock()
	allowCache.byClient = entries
	allowCache.Unlock()
	return nil
}

func publishSecurityEvent(clientName string, kind string, payload map[string]any) {
	if !shouldPublishSecurityEvent(clientName, kind, time.Now()) {
		return
	}
	realtime.Publish(realtime.TopicSecurityEvent, payload)
}

func shouldPublishSecurityEvent(clientName string, kind string, now time.Time) bool {
	key := clientName + "|" + kind
	securityEvents.Lock()
	defer securityEvents.Unlock()
	if last, ok := securityEvents.lastEmittedAt[key]; ok && now.Sub(last) < securityEventDebounce {
		return false
	}
	securityEvents.lastEmittedAt[key] = now
	for eventKey, last := range securityEvents.lastEmittedAt {
		if now.Sub(last) > securityEventMaxMapAge {
			delete(securityEvents.lastEmittedAt, eventKey)
		}
	}
	return true
}

func Flush() error {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	pending.Lock()
	snapshot := pending.byClient
	pending.byClient = map[string]map[string]pendingIP{}
	pending.Unlock()
	if len(snapshot) == 0 {
		return nil
	}
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	if err := flushSnapshot(tx, snapshot); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func FlushTo(tx *gorm.DB) error {
	pending.Lock()
	snapshot := pending.byClient
	pending.byClient = map[string]map[string]pendingIP{}
	pending.Unlock()
	if len(snapshot) == 0 {
		return nil
	}
	return flushSnapshot(tx, snapshot)
}

func flushSnapshot(tx *gorm.DB, snapshot map[string]map[string]pendingIP) error {
	rows := make([]model.ClientIP, 0)
	lastSeenByClient := make(map[string]int64, len(snapshot))
	for clientName, ips := range snapshot {
		for ipHash, pendingIP := range ips {
			if pendingIP.lastSeen > lastSeenByClient[clientName] {
				lastSeenByClient[clientName] = pendingIP.lastSeen
			}
			rows = append(rows, model.ClientIP{
				ClientName: clientName,
				IPHash:     ipHash,
				IPDisplay:  pendingIP.display,
				FirstSeen:  pendingIP.lastSeen,
				LastSeen:   pendingIP.lastSeen,
			})
			cacheAddIP(clientName, ipHash)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	// One batched upsert replaces the former per-IP SELECT + INSERT/UPDATE (an
	// N+1 that ran every 10s). Legacy ip-only rows were given an ip_hash by
	// migration 1.5, so the (client_name, ip_hash) conflict target always
	// matches; first_seen is preserved while last_seen/ip_display are refreshed.
	batch := database.SafeSQLiteBatchSize(tx, &model.ClientIP{})
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "client_name"}, {Name: "ip_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_seen", "ip_display"}),
	}).CreateInBatches(&rows, batch).Error; err != nil {
		return err
	}
	// Refresh each active client's last_online and last_ip_count; the count is
	// folded into the same UPDATE via a correlated subquery (was a separate COUNT).
	for clientName, lastSeen := range lastSeenByClient {
		if err := tx.Model(model.Client{}).Where("name = ?", clientName).Updates(map[string]interface{}{
			"last_online":   lastSeen,
			"last_ip_count": gorm.Expr("(SELECT COUNT(*) FROM client_ips WHERE client_name = ?)", clientName),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func History(clientName string, limit int) ([]model.ClientIP, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows := make([]model.ClientIP, 0)
	err := database.GetDB().Model(model.ClientIP{}).
		Where("client_name = ?", clientName).
		Order("last_seen desc").
		Limit(limit).
		Find(&rows).Error
	if err == nil {
		prepareHistoryRows(rows)
	}
	return rows, err
}

func Clear(clientName string) error {
	db := database.GetDB()
	if err := db.Where("client_name = ?", clientName).Delete(&model.ClientIP{}).Error; err != nil {
		return err
	}
	invalidateCache(clientName)
	return db.Model(model.Client{}).Where("name = ?", clientName).Updates(map[string]interface{}{
		"last_ip_count": 0,
	}).Error
}

func cachedClient(clientName string, now time.Time) (allowCacheEntry, bool) {
	allowCache.Lock()
	defer allowCache.Unlock()
	if entry, ok := allowCache.byClient[clientName]; ok && now.Before(entry.expiresAt) {
		return cloneCacheEntry(entry), true
	}
	delete(allowCache.byClient, clientName)
	return allowCacheEntry{}, false
}

// loadErrLog throttles fail-open DB-error logging so a database outage cannot
// flood the log: ip-limit checks fail open (allow) on a DB error, and without a
// throttle every refresh during the outage would emit a line.
var loadErrLog = struct {
	sync.Mutex
	last time.Time
}{}

func logLoadCacheError(context string, err error) {
	loadErrLog.Lock()
	defer loadErrLog.Unlock()
	if !loadErrLog.last.IsZero() && time.Since(loadErrLog.last) < 30*time.Second {
		return
	}
	loadErrLog.last = time.Now()
	logger.Warning("ipmonitor: ip-limit ", context, " lookup failed; failing open (allowing): ", err)
}

func loadCacheEntry(clientName string, now time.Time) (allowCacheEntry, bool) {
	db := database.GetDB()
	if db == nil {
		return allowCacheEntry{}, false
	}
	var client model.Client
	if err := db.Model(model.Client{}).Select("enable, limit_ip, ip_limit_mode").Where("name = ?", clientName).First(&client).Error; err != nil {
		if !database.IsNotFound(err) {
			logLoadCacheError("client", err)
		}
		return allowCacheEntry{}, false
	}
	if !client.Enable {
		return allowCacheEntry{}, false
	}
	entry := allowCacheEntry{
		limit:     client.LimitIP,
		mode:      client.IPLimitMode,
		ips:       map[string]struct{}{},
		expiresAt: now.Add(allowCacheTTL),
	}
	rows := make([]model.ClientIP, 0)
	if err := db.Model(model.ClientIP{}).Select("ip, ip_hash").Where("client_name = ?", clientName).Find(&rows).Error; err != nil {
		logLoadCacheError("client_ips", err)
		return allowCacheEntry{}, false
	}
	for _, row := range rows {
		ipHash := row.IPHash
		if ipHash == "" {
			ipHash = hashLegacyIPValue(row.IP)
		}
		if ipHash != "" {
			entry.ips[ipHash] = struct{}{}
		}
	}
	return entry, true
}

type activeEnforceCacheRow struct {
	ClientName  string
	LimitIP     int
	IPLimitMode string
	IP          sql.NullString
	IPHash      sql.NullString
}

func loadActiveEnforceEntries(db *gorm.DB, now time.Time) (map[string]allowCacheEntry, error) {
	rows := make([]activeEnforceCacheRow, 0)
	err := db.Raw(`
		SELECT
			clients.name AS client_name,
			clients.limit_ip,
			clients.ip_limit_mode,
			client_ips.ip,
			client_ips.ip_hash
		FROM clients
		LEFT JOIN client_ips ON client_ips.client_name = clients.name
		WHERE clients.enable = true
			AND clients.ip_limit_mode = ?
			AND clients.limit_ip > 0
		ORDER BY clients.name
	`, ModeEnforce).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	entries := make(map[string]allowCacheEntry)
	for _, row := range rows {
		entry, ok := entries[row.ClientName]
		if !ok {
			entry = allowCacheEntry{
				limit:     row.LimitIP,
				mode:      row.IPLimitMode,
				ips:       map[string]struct{}{},
				expiresAt: now.Add(allowCacheTTL),
			}
		}
		ipHash := ""
		if row.IPHash.Valid {
			ipHash = row.IPHash.String
		}
		if ipHash == "" && row.IP.Valid {
			ipHash = hashLegacyIPValue(row.IP.String)
		}
		if ipHash != "" {
			entry.ips[ipHash] = struct{}{}
		}
		entries[row.ClientName] = entry
	}
	return entries, nil
}

func refreshClientAsync(clientName string) {
	allowCacheRefresh.Lock()
	if _, ok := allowCacheRefresh.inFlight[clientName]; ok {
		allowCacheRefresh.Unlock()
		return
	}
	allowCacheRefresh.inFlight[clientName] = struct{}{}
	allowCacheRefresh.Unlock()

	go func() {
		defer func() {
			allowCacheRefresh.Lock()
			delete(allowCacheRefresh.inFlight, clientName)
			allowCacheRefresh.Unlock()
		}()
		refreshClient(clientName, time.Now())
	}()
}

func refreshClient(clientName string, now time.Time) bool {
	entry, ok := loadCacheEntry(clientName, now)
	allowCache.Lock()
	defer allowCache.Unlock()
	if !ok {
		delete(allowCache.byClient, clientName)
		return false
	}
	allowCache.byClient[clientName] = entry
	return true
}

func cloneCacheEntry(entry allowCacheEntry) allowCacheEntry {
	clone := allowCacheEntry{
		limit:     entry.limit,
		mode:      entry.mode,
		ips:       make(map[string]struct{}, len(entry.ips)),
		expiresAt: entry.expiresAt,
	}
	for ip := range entry.ips {
		clone.ips[ip] = struct{}{}
	}
	return clone
}

func cacheAddIP(clientName string, ip string) {
	allowCache.Lock()
	defer allowCache.Unlock()
	entry, ok := allowCache.byClient[clientName]
	if !ok || time.Now().After(entry.expiresAt) {
		return
	}
	if entry.ips == nil {
		entry.ips = map[string]struct{}{}
	}
	entry.ips[ip] = struct{}{}
	allowCache.byClient[clientName] = entry
}

func invalidateCache(clientName string) {
	allowCache.Lock()
	defer allowCache.Unlock()
	delete(allowCache.byClient, clientName)
}

func InvalidateAllCache() {
	allowCache.Lock()
	defer allowCache.Unlock()
	allowCache.byClient = map[string]allowCacheEntry{}
}

func recordIPFields(ip string) (string, *string, bool) {
	ipHash, err := hashIP(ip)
	if err != nil {
		return "", nil, false
	}
	showRaw, err := getIPShowRaw(time.Now())
	if err != nil || !showRaw {
		return ipHash, nil, true
	}
	display := ip
	return ipHash, &display, true
}

func hashIP(ip string) (string, error) {
	salt, err := getInstallSalt()
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, _ = h.Write(salt)
	_, _ = h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func getInstallSalt() ([]byte, error) {
	ipHashSalt.Lock()
	defer ipHashSalt.Unlock()
	if len(ipHashSalt.value) > 0 {
		salt := make([]byte, len(ipHashSalt.value))
		copy(salt, ipHashSalt.value)
		return salt, nil
	}
	if database.GetDB() == nil {
		return nil, errors.New("database is not initialized")
	}
	var setting model.Setting
	err := database.GetDB().Model(model.Setting{}).Where("key = ?", "installSalt").First(&setting).Error
	if database.IsNotFound(err) {
		setting = model.Setting{Key: "installSalt", Value: common.Random(32)}
		err = database.GetDB().Create(&setting).Error
	}
	if err != nil {
		return nil, err
	}
	salt := []byte(setting.Value)
	ipHashSalt.value = append([]byte(nil), salt...)
	return append([]byte(nil), salt...), nil
}

func getIPShowRaw(now time.Time) (bool, error) {
	ipPrivacySettings.Lock()
	defer ipPrivacySettings.Unlock()
	if now.Before(ipPrivacySettings.expiresAt) {
		return ipPrivacySettings.showRaw, nil
	}
	if database.GetDB() == nil {
		ipPrivacySettings.showRaw = false
		ipPrivacySettings.expiresAt = now.Add(allowCacheTTL)
		return false, nil
	}
	var setting model.Setting
	err := database.GetDB().Model(model.Setting{}).Where("key = ?", "ipShowRaw").First(&setting).Error
	if database.IsNotFound(err) {
		ipPrivacySettings.showRaw = false
		ipPrivacySettings.expiresAt = now.Add(allowCacheTTL)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	showRaw, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return false, err
	}
	ipPrivacySettings.showRaw = showRaw
	ipPrivacySettings.expiresAt = now.Add(allowCacheTTL)
	return showRaw, nil
}

func prepareHistoryRows(rows []model.ClientIP) {
	showRaw, err := getIPShowRaw(time.Now())
	if err != nil {
		showRaw = false
	}
	for i := range rows {
		display := maskedIP(rows[i])
		if showRaw {
			if rows[i].IPDisplay != nil && *rows[i].IPDisplay != "" {
				display = *rows[i].IPDisplay
			} else if rows[i].IPHash == "" && !looksLikeSHA256Hex(rows[i].IP) {
				display = rows[i].IP
			}
		}
		rows[i].IP = display
		rows[i].IPHash = ""
		rows[i].IPDisplay = nil
	}
}

func maskedIP(row model.ClientIP) string {
	ipHash := row.IPHash
	if ipHash == "" {
		ipHash = hashLegacyIPValue(row.IP)
	}
	if len(ipHash) < ipMaskPrefix {
		return "masked"
	}
	return "masked:" + ipHash[:ipMaskPrefix]
}

func hashLegacyIPValue(ip string) string {
	if looksLikeSHA256Hex(ip) {
		return ip
	}
	ipHash, err := hashIP(ip)
	if err != nil {
		return ""
	}
	return ipHash
}

func looksLikeSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
