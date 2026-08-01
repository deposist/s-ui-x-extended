package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/realtime"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/redact"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type ApiService struct {
	Runtime *service.Runtime
	service.SettingService
	service.UserService
	service.ConfigService
	service.ClientService
	service.TlsService
	service.InboundService
	service.OutboundService
	service.EndpointService
	service.ProviderService
	service.ServicesService
	service.PanelService
	service.StatsService
	service.ServerService
	service.AuditService
	service.ObservabilityService
	service.TelegramService
	service.VersionService
	service.PanelUpdateService
	service.DoctorService
	service.RuleSetAssetService
}

type Option func(*ApiService)

func WithRuntime(runtime *service.Runtime) Option {
	return func(a *ApiService) {
		if runtime != nil {
			a.Runtime = runtime
		}
	}
}

func NewApiService(options ...Option) ApiService {
	a := ApiService{
		Runtime: service.DefaultRuntime(),
	}
	for _, option := range options {
		if option != nil {
			option(&a)
		}
	}
	a.bindRuntime()
	return a
}

func (a *ApiService) bindRuntime() {
	runtime := a.Runtime
	if runtime == nil {
		runtime = service.DefaultRuntime()
		a.Runtime = runtime
	}
	a.UserService = service.UserService{Runtime: runtime}
	a.ConfigService = *service.NewConfigServiceWithRuntime(runtime)
	a.ClientService = service.ClientService{Runtime: runtime}
	a.TlsService = service.TlsService{
		Runtime:         runtime,
		InboundService:  service.InboundService{Runtime: runtime, ClientService: service.ClientService{Runtime: runtime}},
		ServicesService: service.ServicesService{Runtime: runtime},
	}
	a.InboundService = service.InboundService{Runtime: runtime, ClientService: service.ClientService{Runtime: runtime}}
	a.ServicesService = service.ServicesService{Runtime: runtime}
	a.PanelService = service.PanelService{Runtime: runtime}
	a.StatsService = service.StatsService{Runtime: runtime}
	a.ServerService = service.ServerService{Runtime: runtime}
	a.AuditService = service.AuditService{Runtime: runtime}
	a.ObservabilityService = service.ObservabilityService{
		ServerService: service.ServerService{Runtime: runtime},
	}
	a.TelegramService = service.TelegramService{Runtime: runtime}
	a.PanelUpdateService = service.PanelUpdateService{Runtime: runtime}
	a.DoctorService = service.DoctorService{Runtime: runtime}
}

const maxDatabaseImportBytes = 64 << 20

type memoryMultipartFile struct {
	*bytes.Reader
}

func (f memoryMultipartFile) Close() error {
	return nil
}

func (a *ApiService) LoadData(c *gin.Context) {
	data, err := a.getData(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, nil)
}

func (a *ApiService) getData(c *gin.Context) (interface{}, error) {
	// One-time lazy backfill of sudoku/mieru client links, using the host the
	// operator reached the panel on. Startup has no request host and
	// settings.webDomain is usually blank, so this is where existing clients get
	// their links (and QR) without a manual re-save. Runs once per process.
	a.ClientService.RegenerateMissingLocalLinksOnce(getHostname(c))

	data := make(map[string]interface{}, 0)
	// Capture the acknowledgement cursor before checking and reading the
	// snapshot. A concurrent commit will then remain newer than this response and
	// be fetched next time rather than being acknowledged before it was read.
	revision := a.ConfigService.CurrentRevision()
	lu := c.Query("lu")
	isUpdated, err := a.ConfigService.CheckChanges(lu)
	if err != nil {
		return "", err
	}
	data["revision"] = revision
	onlines, err := a.StatsService.GetOnlines()

	sysInfo := a.ServerService.GetSingboxInfo()
	if sysInfo["running"] == false {
		logs := a.ServerService.GetLogs("1", "debug")
		if len(logs) > 0 {
			data["lastLog"] = logs[0]
		}
	}

	if err != nil {
		return "", err
	}
	if isUpdated {
		var loadSettings service.PanelLoadSettings
		var clients, tlsConfigs, inbounds, outbounds, endpoints, services any
		var group errgroup.Group
		group.Go(func() error {
			settings, err := a.SettingService.LoadPanelSettingsForData(getHostname(c))
			loadSettings = settings
			return err
		})
		group.Go(func() error {
			result, err := a.ClientService.GetAll()
			clients = result
			return err
		})
		group.Go(func() error {
			result, err := a.TlsService.GetAll()
			tlsConfigs = result
			return err
		})
		group.Go(func() error {
			result, err := a.InboundService.GetAll()
			inbounds = result
			return err
		})
		group.Go(func() error {
			result, err := a.OutboundService.GetAll()
			outbounds = result
			return err
		})
		group.Go(func() error {
			result, err := a.EndpointService.GetAll()
			endpoints = result
			return err
		})
		group.Go(func() error {
			result, err := a.ServicesService.GetAll()
			services = result
			return err
		})
		if err := group.Wait(); err != nil {
			return "", err
		}
		data["config"] = json.RawMessage(loadSettings.Config)
		data["clients"] = clients
		data["tls"] = tlsConfigs
		data["inbounds"] = inbounds
		data["outbounds"] = outbounds
		data["endpoints"] = endpoints
		data["services"] = services
		data["subURI"] = loadSettings.SubURI
		if loadSettings.SubJsonURI != "" {
			data["subJsonURI"] = loadSettings.SubJsonURI
		}
		if loadSettings.SubClashURI != "" {
			data["subClashURI"] = loadSettings.SubClashURI
		}
		data["enableTraffic"] = loadSettings.TrafficAge > 0
		data["onlines"] = onlines
	} else {
		data["onlines"] = onlines
	}

	return data, nil
}

func (a *ApiService) LoadPartialData(c *gin.Context, objs []string) error {
	data := make(map[string]interface{}, 0)
	id := c.Query("id")

	for _, obj := range objs {
		switch obj {
		case "inbounds":
			inbounds, err := a.InboundService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = inbounds
		case "outbounds":
			outbounds, err := a.OutboundService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = outbounds
		case "endpoints":
			endpoints, err := a.EndpointService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = endpoints
		case "providers":
			providers, err := a.ProviderService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = providers
		case "services":
			services, err := a.ServicesService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = services
		case "tls":
			tlsConfigs, err := a.TlsService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = tlsConfigs
		case "clients":
			clients, err := a.ClientService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = clients
		case "config":
			config, err := a.SettingService.GetConfig()
			if err != nil {
				return err
			}
			data[obj] = json.RawMessage(config)
		case "settings":
			settings, err := a.SettingService.GetAllSetting()
			if err != nil {
				return err
			}
			data[obj] = settings
		}
	}

	jsonObj(c, data, nil)
	return nil
}

func (a *ApiService) GetUsers(c *gin.Context) {
	users, err := a.UserService.GetUsers()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	loginUser := GetLoginUser(c)
	result := make([]gin.H, 0, len(*users))
	for _, user := range *users {
		result = append(result, gin.H{
			"id":        user.Id,
			"username":  user.Username,
			"lastLogin": user.LastLogins,
			"isCurrent": user.Username == loginUser,
		})
	}
	jsonObj(c, result, nil)
}

func (a *ApiService) GetSettings(c *gin.Context) {
	data, err := a.SettingService.GetAllSetting()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStats(c *gin.Context) {
	resource := c.Query("resource")
	tag := c.Query("tag")
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		limit = 100
	}
	data, err := a.StatsService.GetStats(resource, tag, limit)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetTrafficStats(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	buckets, _ := strconv.Atoi(c.Query("buckets"))
	endTime, _ := strconv.ParseInt(c.Query("end"), 10, 64)
	data, err := a.StatsService.GetInboundTrafficSummary(limit, buckets, endTime)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStatus(c *gin.Context) {
	request := c.Query("r")
	result := a.ServerService.GetStatus(request)
	jsonObj(c, result, nil)
}

func (a *ApiService) GetOnlines(c *gin.Context) {
	onlines, err := a.StatsService.GetOnlines()
	jsonObj(c, onlines, err)
}

func (a *ApiService) GetCapabilities(c *gin.Context) {
	jsonObj(c, capabilities.BuildAPIView(), nil)
}

func (a *ApiService) GetLogs(c *gin.Context) {
	count := c.Query("count")
	if count == "" {
		count = c.Query("c")
	}
	level := c.Query("level")
	if level == "" {
		level = c.Query("l")
	}
	logs, err := a.ServerService.GetLogsFiltered(count, level, c.Query("source"), c.Query("filter"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "logs: " + redact.String(err.Error())})
		return
	}
	jsonObj(c, logs, nil)
}

func (a *ApiService) CheckChanges(c *gin.Context) {
	actor := c.Query("a")
	chngKey := c.Query("k")
	count := c.Query("c")
	changes := a.ConfigService.GetChanges(actor, chngKey, count)
	jsonObj(c, changes, nil)
}

func (a *ApiService) GetKeypairs(c *gin.Context) {
	kType := c.Query("k")
	options := c.Query("o")
	keypair := a.ServerService.GenKeypair(kType, options)
	jsonObj(c, keypair, nil)
}

func (a *ApiService) GetDb(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "database", "admin") {
		return
	}
	exclude := c.Query("exclude")
	if c.Query("encryptTelegramBackup") == "true" {
		a.getEncryptedDb(c, exclude)
		return
	}
	backupPath, cleanup, err := database.PrepareDbBackup(exclude)
	if err != nil {
		a.recordAudit(c, requestActor(c), "db_export_failed", "database", service.AuditSeverityWarn, map[string]any{
			"channel": "download",
		})
		jsonMsg(c, "", err)
		return
	}
	defer cleanup()
	// #nosec G304 -- backupPath is a freshly-created local backup file path.
	backupFile, err := os.Open(backupPath)
	if err != nil {
		a.recordAudit(c, requestActor(c), "db_export_failed", "database", service.AuditSeverityWarn, map[string]any{
			"channel": "download",
		})
		jsonMsg(c, "", err)
		return
	}
	defer backupFile.Close()
	a.recordAudit(c, requestActor(c), "db_exported", "database", service.AuditSeverityWarn, map[string]any{
		"channel": "download",
		"exclude": exclude,
	})
	// Real-time alert on config exfiltration (T1530): a full DB export is one of
	// the highest-signal admin-compromise events.
	a.TelegramService.NotifyTelegramEvent("db_exported", map[string]string{
		"actor": requestActor(c),
		"ip":    getRemoteIp(c),
	})
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=s-ui_"+time.Now().Format("20060102-150405")+".db")
	_, _ = io.Copy(c.Writer, backupFile)
}

func (a *ApiService) getEncryptedDb(c *gin.Context, exclude string) {
	hasPassphrase, err := a.SettingService.HasTelegramBackupPassphrase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Msg{
			Success: false,
			Msg:     "backup: settings",
			Obj:     gin.H{"errorClass": "settings"},
		})
		return
	}
	if !hasPassphrase {
		c.JSON(http.StatusBadRequest, Msg{
			Success: false,
			Msg:     "backup: missing_passphrase",
			Obj:     gin.H{"errorClass": "missing_passphrase"},
		})
		return
	}
	db, err := database.GetDb(exclude)
	if err != nil {
		a.recordAudit(c, requestActor(c), "db_export_failed", "database", service.AuditSeverityWarn, map[string]any{
			"channel":   "local_download",
			"encrypted": true,
		})
		jsonMsg(c, "", err)
		return
	}
	payloadSize := len(db)
	defer wipeBytes(db)

	passphrase, err := a.SettingService.GetTelegramBackupPassphraseBytes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Msg{
			Success: false,
			Msg:     "backup: settings",
			Obj:     gin.H{"errorClass": "settings"},
		})
		return
	}
	defer wipeBytes(passphrase)
	if len(passphrase) == 0 {
		c.JSON(http.StatusBadRequest, Msg{
			Success: false,
			Msg:     "backup: missing_passphrase",
			Obj:     gin.H{"errorClass": "missing_passphrase"},
		})
		return
	}

	envelope, err := service.BuildTelegramBackupEnvelope(db, passphrase)
	wipeBytes(passphrase)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Msg{
			Success: false,
			Msg:     "backup: encryption_failed",
			Obj:     gin.H{"errorClass": "encryption_failed"},
		})
		return
	}
	wipeBytes(db)

	a.recordAudit(c, requestActor(c), "tg_backup_manual_encrypted", "database", service.AuditSeverityInfo, map[string]any{
		"channel":           "local_download",
		"payloadSizeBytes":  int64(payloadSize),
		"envelopeSizeBytes": int64(len(envelope)),
		"excludedTables":    database.ParseBackupExcludes(exclude),
	})
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=s-ui_"+time.Now().Format("20060102-150405")+".db.aes")
	_, _ = c.Writer.Write(envelope)
}

func (a *ApiService) Login(c *gin.Context) {
	remoteIP := getRemoteIp(c)
	username := c.Request.FormValue("user")
	userKey := loginRateLimitUserKey(username)
	// Two independent throttles: per source IP (one attacker host) and per
	// username (a distributed brute-force on one account from rotating IPs,
	// which the per-IP limit alone cannot stop).
	if err := checkLoginRateLimit(remoteIP); err != nil {
		a.recordAudit(c, username, "login_blocked", "auth", service.AuditSeverityWarn, map[string]any{
			"reason": "rate_limit_ip",
		})
		// Real-time alert on the lockout transition (T1110): brute-force reaching
		// the per-IP block is a high-signal admin-compromise indicator.
		a.TelegramService.NotifyTelegramEvent("login_blocked", telegramRequestFields(c))
		jsonMsg(c, "", err)
		return
	}
	// Per-username throttle is a tarpit (escalating, capped delay), never a hard
	// block — so a distributed attacker burning failures from rotating IPs
	// cannot lock a known admin out of their own panel. The per-IP hard block
	// above remains the primary brute-force defence.
	if delay := loginUsernameTarpitDelay(userKey); delay > 0 {
		select {
		case <-time.After(delay):
		case <-c.Request.Context().Done():
			return
		}
	}
	loginUser, forcePasswordReset, err := a.UserService.Login(username, c.Request.FormValue("pass"), remoteIP)
	if err != nil && !errors.Is(err, service.ErrForcePasswordReset) {
		recordLoginFailure(remoteIP)
		recordLoginFailure(userKey)
		a.recordAudit(c, username, "login_failed", "auth", service.AuditSeverityWarn, map[string]any{
			"reason": redact.String(err.Error()),
		})
		a.TelegramService.NotifyTelegramEvent("login_failed", telegramRequestFields(c))
		jsonMsg(c, "", err)
		return
	}
	resetLoginFailures(remoteIP)
	resetLoginFailures(userKey)

	sessionMaxAge, err := a.SettingService.GetSessionMaxAge()
	if err != nil {
		logger.Infof("Unable to get session's max age from DB")
	}

	sessionGeneration, err := a.SettingService.GetSessionGeneration()
	if err != nil {
		logger.Warning("unable to get session generation:", err)
	}

	if forcePasswordReset {
		err = SetForcePasswordResetUser(c, loginUser, sessionMaxAge, sessionGeneration)
	} else {
		err = SetLoginUser(c, loginUser, sessionMaxAge, sessionGeneration)
	}
	if err == nil {
		if forcePasswordReset {
			logger.Info("user ", loginUser, " must change password")
			a.recordAudit(c, loginUser, "password_reset_required", "auth", service.AuditSeverityWarn, nil)
			c.JSON(http.StatusOK, Msg{Success: false, Msg: "", Obj: gin.H{"forcePasswordReset": true, "username": loginUser}})
			return
		}
		logger.Info("user ", loginUser, " login success")
		a.recordAudit(c, loginUser, "login_success", "auth", service.AuditSeverityInfo, nil)
		a.TelegramService.NotifyTelegramEvent("login_success", map[string]string{
			"user": loginUser,
			"ip":   remoteIP,
		})
	} else {
		logger.Warning("login failed: ", err)
		a.recordAudit(c, loginUser, "login_session_failed", "auth", service.AuditSeverityWarn, map[string]any{
			"reason": redact.String(err.Error()),
		})
	}

	jsonMsg(c, "", nil)
}

func (a *ApiService) ChangePass(c *gin.Context) {
	oldPass := c.Request.FormValue("oldPass")
	newUsername := c.Request.FormValue("newUsername")
	newPass := c.Request.FormValue("newPass")
	// Bind the change to the authenticated session user; never trust a target id
	// from the request, so one admin cannot change another admin's credentials.
	currentUser := GetLoginUser(c)
	err := a.UserService.ChangePass(currentUser, oldPass, newUsername, newPass)
	if err == nil {
		// ChangePass commits before returning. Consume the bootstrap credential only
		// after that durable credential rotation; a failed change leaves it intact.
		if cleanupErr := database.ConsumeInitialAdminPasswordIfBootstrapCredential(currentUser, oldPass); cleanupErr != nil {
			logger.Warning("unable to remove initial admin credential:", cleanupErr)
		}
		logger.Info("change user credentials success")
		a.recordAudit(c, currentUser, "admin_credentials_changed", "admin", service.AuditSeverityWarn, map[string]any{
			"newUsername": newUsername,
		})
		// Rotate the session generation so every OTHER web session and all WS
		// tokens (including any minted under the old credentials) are invalidated,
		// then re-establish only THIS session under the new generation so the
		// admin who changed the password is not logged out of their own session.
		if newGen, rerr := a.SettingService.RotateSessionGeneration(); rerr != nil {
			logger.Warning("session rotation after credential change failed:", rerr)
		} else {
			sessionMaxAge, _ := a.SettingService.GetSessionMaxAge()
			if serr := SetLoginUser(c, newUsername, sessionMaxAge, newGen); serr != nil {
				logger.Warning("re-establishing session after credential change failed:", serr)
			}
		}
		jsonMsg(c, "save", nil)
	} else {
		logger.Warning("change user credentials failed:", err)
		jsonMsg(c, "", err)
	}
}

func (a *ApiService) AddAdmin(c *gin.Context) {
	loginUser := GetLoginUser(c)
	user, err := a.UserService.AddUser(
		loginUser,
		c.Request.FormValue("currentPass"),
		c.Request.FormValue("username"),
		c.Request.FormValue("password"),
	)
	if err == nil {
		logger.Info("admin user created successfully")
		a.recordAudit(c, loginUser, "admin_created", "admin", service.AuditSeverityWarn, map[string]any{
			"targetUserId": user.Id,
			"username":     user.Username,
		})
		jsonMsgObj(c, "add", gin.H{
			"id":        user.Id,
			"username":  user.Username,
			"lastLogin": user.LastLogins,
			"isCurrent": false,
		}, nil)
	} else {
		logger.Warning("create admin user failed:", err)
		jsonMsg(c, "", err)
	}
}

func (a *ApiService) DeleteAdmin(c *gin.Context) {
	loginUser := GetLoginUser(c)
	result, err := a.UserService.DeleteUser(
		loginUser,
		c.Request.FormValue("currentPass"),
		c.Request.FormValue("id"),
	)
	if err == nil {
		logger.Info("admin user deleted successfully")
		a.recordAudit(c, loginUser, "admin_deleted", "admin", service.AuditSeverityWarn, map[string]any{
			"targetUserId":      result.User.Id,
			"username":          result.User.Username,
			"deletedTokenCount": result.DeletedTokenCount,
		})
		jsonMsg(c, "del", nil)
	} else {
		logger.Warning("delete admin user failed:", err)
		jsonMsg(c, "", err)
	}
}

func (a *ApiService) Save(c *gin.Context, loginUser string) {
	hostname := getHostname(c)
	obj := c.Request.FormValue("object")
	act := c.Request.FormValue("action")
	data := c.Request.FormValue("data")
	initUsers := c.Request.FormValue("initUsers")

	// Authoritative duplicate-create guard: an identical create resubmitted
	// within a short window (double-click / client double-send / proxy replay)
	// is skipped so it cannot insert a second row. Only creation actions on
	// entity objects are guarded; the claim is released below if the save fails
	// so a failed create can be retried immediately.
	dedupKey := ""
	if isDedupableSave(obj, act) {
		dedupKey = saveDedupKey(loginUser, obj, act, data)
		if !saveDedup.claim(dedupKey, time.Now().UnixNano()) {
			logger.Warning("save: skipped duplicate '", obj, "' create within dedup window")
			if err := a.LoadPartialData(c, []string{obj}); err != nil {
				jsonMsg(c, obj, err)
			}
			return
		}
	}

	subscriptionPathBefore := a.subscriptionPathSnapshot(obj, data)
	objs, err := a.ConfigService.Save(obj, act, json.RawMessage(data), initUsers, loginUser, hostname)
	if err != nil {
		if dedupKey != "" {
			saveDedup.release(dedupKey)
		}
		invalidSettingKey := false
		if obj == "settings" {
			event := "settings_save_rejected"
			if strings.Contains(err.Error(), "invalid setting key:") {
				event = "settings_save_rejected_key"
				invalidSettingKey = true
			}
			a.recordAudit(c, loginUser, event, "settings", service.AuditSeverityWarn, map[string]any{
				"reason": redact.String(err.Error()),
			})
		}
		if invalidSettingKey {
			c.JSON(http.StatusBadRequest, Msg{Success: false, Msg: "save: " + redact.String(err.Error())})
			return
		}
		jsonMsg(c, "save", err)
		return
	}
	// Save (incl. any synchronous core restart) succeeded and the row is
	// committed; keep deduplicating an identical resubmit for the window.
	if dedupKey != "" {
		saveDedup.complete(dedupKey, time.Now().UnixNano())
	}
	if obj == "settings" {
		a.recordAudit(c, loginUser, "settings_save_succeeded", "settings", service.AuditSeverityInfo, map[string]any{
			"action": act,
		})
	}
	a.auditSubscriptionPathChanges(c, loginUser, subscriptionPathBefore)
	err = a.LoadPartialData(c, objs)
	if err != nil {
		jsonMsg(c, obj, err)
	}
}

func (a *ApiService) subscriptionPathSnapshot(obj string, data string) map[string]string {
	if obj != "settings" {
		return nil
	}
	var settings map[string]string
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		return nil
	}

	before := make(map[string]string, 3)
	if _, ok := settings["subPath"]; ok {
		if path, err := a.SettingService.GetSubPath(); err == nil {
			before["subPath"] = path
		}
	}
	if _, ok := settings["subJsonPath"]; ok {
		if path, err := a.SettingService.GetSubJsonPath(); err == nil {
			before["subJsonPath"] = path
		}
	}
	if _, ok := settings["subClashPath"]; ok {
		if path, err := a.SettingService.GetSubClashPath(); err == nil {
			before["subClashPath"] = path
		}
	}
	if len(before) == 0 {
		return nil
	}
	return before
}

func (a *ApiService) auditSubscriptionPathChanges(c *gin.Context, actor string, before map[string]string) {
	if len(before) == 0 {
		return
	}
	changed := map[string]map[string]string{}
	for key, oldPath := range before {
		var newPath string
		var err error
		switch key {
		case "subPath":
			newPath, err = a.SettingService.GetSubPath()
		case "subJsonPath":
			newPath, err = a.SettingService.GetSubJsonPath()
		case "subClashPath":
			newPath, err = a.SettingService.GetSubClashPath()
		default:
			continue
		}
		if err != nil || newPath == oldPath {
			continue
		}
		changed[key] = map[string]string{
			"old": oldPath,
			"new": newPath,
		}
	}
	if len(changed) == 0 {
		return
	}
	a.recordAudit(c, actor, "sub_path_changed", "subscription", service.AuditSeverityWarn, map[string]any{
		"paths":           changed,
		"restartRequired": true,
	})
}

func (a *ApiService) RestartApp(c *gin.Context) {
	err := a.PanelService.RestartPanel(3 * time.Second)
	jsonMsg(c, "restartApp", err)
}

// RegenerateClientLinks rebuilds every client's local links and QR from the host
// the operator reached the panel on. It backs the settings button for clients
// whose links are missing or stale (for example sudoku/mieru links added in a
// later build, or after changing the panel domain).
func (a *ApiService) RegenerateClientLinks(c *gin.Context) {
	count, err := a.ClientService.RegenerateAllClientLinks(getHostname(c))
	jsonMsgObj(c, "regenerateClientLinks", map[string]any{"count": count}, err)
}

func (a *ApiService) RestartSb(c *gin.Context) {
	err := a.ConfigService.RestartCore()
	if err != nil {
		a.TelegramService.NotifyTelegramEvent("core_restart_failed", coreRestartFailedTelegramFields(c, err))
	} else {
		a.TelegramService.NotifyTelegramEvent("core_restarted", nil)
	}
	jsonMsg(c, "restartSb", err)
}

func telegramRequestFields(c *gin.Context) map[string]string {
	return map[string]string{
		"ip":      getRemoteIp(c),
		"ua_hash": hashUserAgent(c.Request.UserAgent()),
		"ts":      time.Now().UTC().Format(time.RFC3339),
	}
}

func hashUserAgent(userAgent string) string {
	sum := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(sum[:])
}

func coreRestartFailedTelegramFields(c *gin.Context, err error) map[string]string {
	fields := telegramRequestFields(c)
	fields["errorClass"] = coreRestartErrorClass(err)
	return fields
}

func coreRestartErrorClass(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ToLower(redact.String(err.Error()))
	switch {
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return "timeout"
	case strings.Contains(message, "permission"), strings.Contains(message, "access is denied"):
		return "permission"
	case strings.Contains(message, "config"), strings.Contains(message, "parse"), strings.Contains(message, "json"):
		return "config"
	default:
		return "failed"
	}
}

func (a *ApiService) LinkConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, _, err := util.GetOutbound(link, 0)
	jsonObj(c, result, err)
}

func (a *ApiService) SubConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, err := util.GetExternalSub(link)
	jsonObj(c, result, err)
}

func (a *ApiService) ImportDb(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "database", "admin") {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDatabaseImportBytes)
	file, _, err := c.Request.FormFile("db")
	if err != nil {
		a.recordAudit(c, requestActor(c), "db_import_failed", "database", service.AuditSeverityWarn, map[string]any{
			"reason": databaseImportErrorClass(err),
		})
		jsonMsg(c, "", err)
		return
	}
	defer file.Close()
	importFile, cleanup, ok := a.prepareDatabaseImportFile(c, file)
	if cleanup != nil {
		defer cleanup()
	}
	if !ok {
		return
	}
	err = database.ImportDB(importFile)
	if err != nil {
		a.recordAudit(c, requestActor(c), "db_import_failed", "database", service.AuditSeverityWarn, map[string]any{
			"reason": databaseImportErrorClass(err),
		})
	} else {
		a.recordAudit(c, requestActor(c), "db_imported", "database", service.AuditSeverityWarn, nil)
	}
	jsonMsg(c, "", err)
}

func (a *ApiService) prepareDatabaseImportFile(c *gin.Context, file multipart.File) (multipart.File, func(), bool) {
	header := make([]byte, len(service.TelegramBackupMagic))
	n, readErr := io.ReadFull(file, header)
	if seekErr := seekMultipartFileStart(file); seekErr != nil {
		a.recordAudit(c, requestActor(c), "db_import_failed", "database", service.AuditSeverityWarn, map[string]any{
			"reason": "failed",
		})
		jsonMsg(c, "", seekErr)
		return nil, nil, false
	}
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		a.recordAudit(c, requestActor(c), "db_import_failed", "database", service.AuditSeverityWarn, map[string]any{
			"reason": "failed",
		})
		jsonMsg(c, "", readErr)
		return nil, nil, false
	}
	if !service.IsTelegramBackupEnvelope(header[:n]) {
		return file, nil, true
	}

	passphraseValue := c.PostForm("telegramBackupPassphrase")
	if passphraseValue == "" {
		passphraseValue = c.PostForm("backupPassphrase")
	}
	passphrase := []byte(passphraseValue)
	defer wipeBytes(passphrase)
	if len(passphrase) == 0 {
		a.respondTelegramBackupRestoreDecryptionFailed(c)
		return nil, nil, false
	}
	envelope, err := io.ReadAll(file)
	if err != nil {
		a.respondTelegramBackupRestoreDecryptionFailed(c)
		return nil, nil, false
	}
	defer wipeBytes(envelope)
	plaintext, err := service.OpenTelegramBackupEnvelope(envelope, passphrase)
	if err != nil {
		a.respondTelegramBackupRestoreDecryptionFailed(c)
		return nil, nil, false
	}
	cleanup := func() {
		wipeBytes(plaintext)
	}
	return memoryMultipartFile{Reader: bytes.NewReader(plaintext)}, cleanup, true
}

func seekMultipartFileStart(file multipart.File) error {
	if _, err := file.Seek(0, 0); err != nil {
		return common.NewErrorf("Error resetting file reader: %v", err)
	}
	return nil
}

func (a *ApiService) respondTelegramBackupRestoreDecryptionFailed(c *gin.Context) {
	a.recordAudit(c, requestActor(c), "tg_backup_restore_failed", "database", service.AuditSeverityWarn, map[string]any{
		"errorClass": "decryption_failed",
	})
	c.JSON(http.StatusBadRequest, Msg{
		Success: false,
		Msg:     "restore: decryption_failed",
		Obj: gin.H{
			"errorClass": "decryption_failed",
		},
	})
}

func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func databaseImportErrorClass(err error) string {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return "too_large"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "format"), strings.Contains(msg, "sqlite"), strings.Contains(msg, "integrity"):
		return "invalid_db"
	default:
		return "failed"
	}
}

func (a *ApiService) RotateSubSecret(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "client", "admin", "write") {
		return
	}
	clientID := c.Query("id")
	clientName, err := a.ClientService.RotateSubSecret(clientID)
	if err == nil {
		a.recordAudit(c, requestActor(c), "sub_secret_rotated", "client", service.AuditSeverityWarn, map[string]any{
			"clientId": clientID,
			"client":   clientName,
		})
		realtime.Publish(realtime.TopicConfigInvalidated, nil)
	}
	jsonMsg(c, "rotateSubSecret", err)
}

func (a *ApiService) Logout(c *gin.Context) {
	loginUser := GetLoginUser(c)
	if loginUser != "" {
		logger.Infof("user %s logout", loginUser)
		a.recordAudit(c, loginUser, "logout", "auth", service.AuditSeverityInfo, nil)
	}
	ClearSession(c)
	jsonMsg(c, "", nil)
}

func (a *ApiService) LogoutAllAdmins(c *gin.Context) {
	loginUser := GetLoginUser(c)
	_, err := a.SettingService.RotateSessionGeneration()
	if err == nil {
		if loginUser != "" {
			logger.Infof("user %s logged out all admin web sessions", loginUser)
		}
		a.recordAudit(c, loginUser, "logout_all_admins", "auth", service.AuditSeverityWarn, nil)
		a.TelegramService.NotifyTelegramEvent("logout_all_admins", map[string]string{
			"user": loginUser,
		})
		ClearSession(c)
	}
	jsonMsg(c, "logoutAllAdmins", err)
}

func (a *ApiService) LoadTokens() ([]byte, error) {
	return a.UserService.LoadTokens()
}

func (a *ApiService) GetTokens(c *gin.Context) {
	loginUser := GetLoginUser(c)
	tokens, err := a.UserService.GetUserTokens(loginUser)
	jsonObj(c, tokens, err)
}

func (a *ApiService) AddToken(c *gin.Context) {
	loginUser := GetLoginUser(c)
	expiry := c.Request.FormValue("expiry")
	expiryInt, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	desc := c.Request.FormValue("desc")
	scope := c.DefaultPostForm("scope", "admin")
	token, err := a.UserService.AddToken(loginUser, expiryInt, desc, scope)
	if err == nil {
		a.recordAudit(c, loginUser, "api_token_created", "api_token", service.AuditSeverityWarn, map[string]any{
			"desc":   desc,
			"expiry": expiryInt,
			"scope":  scope,
		})
	}
	jsonObj(c, token, err)
}

func (a *ApiService) DeleteToken(c *gin.Context) {
	tokenId := c.Request.FormValue("id")
	err := a.UserService.DeleteToken(GetLoginUser(c), tokenId)
	if err == nil {
		a.recordAudit(c, GetLoginUser(c), "api_token_deleted", "api_token", service.AuditSeverityWarn, map[string]any{
			"id": tokenId,
		})
	}
	jsonMsg(c, "", err)
}

func (a *ApiService) SetTokenEnabled(c *gin.Context) {
	id := c.Request.FormValue("id")
	enabled, err := strconv.ParseBool(c.Request.FormValue("enabled"))
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	err = a.UserService.SetTokenEnabled(GetLoginUser(c), id, enabled)
	if err == nil {
		a.recordAudit(c, GetLoginUser(c), "api_token_enabled_changed", "api_token", service.AuditSeverityWarn, map[string]any{
			"id":      id,
			"enabled": enabled,
		})
	}
	jsonMsg(c, "save", err)
}

func (a *ApiService) GetSingboxConfig(c *gin.Context) {
	rawConfig, err := a.ConfigService.GetConfig("")
	if err != nil {
		c.Status(400)
		_, _ = c.Writer.WriteString(redact.String(err.Error()))
		return
	}
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=config_"+time.Now().Format("20060102-150405")+".json")
	_, _ = c.Writer.Write(*rawConfig)
}

func (a *ApiService) GetCheckOutbound(c *gin.Context) {
	tag := c.Query("tag")
	link := c.Query("link")
	// A user-supplied custom test target is an SSRF vector: the server fetches it
	// through the selected outbound. Validate it exactly like the plural
	// CheckOutbounds endpoint. An empty link means "use the core's built-in
	// default target" (the normal UI call passes only tag) and is safe.
	if link != "" {
		if err := validateOutboundCheckTarget(c.Request.Context(), link); err != nil {
			jsonMsg(c, "checkOutbound", err)
			return
		}
	}
	result := a.ConfigService.CheckOutbound(tag, link)
	service.RefreshProviderHealth(time.Now())
	jsonObj(c, result, nil)
}

// GetGroupPreview resolves the static and provider-backed members of a group
// outbound for the UI's group editor preview. It does not modify anything.
func (a *ApiService) GetGroupPreview(c *gin.Context) {
	tag := c.Query("tag")
	if tag == "" {
		jsonMsg(c, "groupPreview", common.NewError("missing query parameter: tag"))
		return
	}
	preview, err := a.ConfigService.GroupPreview(tag)
	jsonObj(c, preview, err)
}
