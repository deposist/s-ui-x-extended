package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	stdatomic "sync/atomic"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/realtime"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const (
	wsTokenTTL    = 60 * time.Second
	wsCloseAuth   = websocket.StatusCode(4401)
	maxWSPerUser  = 5
	maxWSPerIP    = 20
	wsQueueSize   = 16
	wsSubprotocol = "sui.realtime"
	wsTokenPrefix = "sui.token."

	defaultWSPingInterval = 25 * time.Second
	defaultWSPingTimeout  = 5 * time.Second

	wsTokenSweepInterval = time.Minute
	maxWSTokens          = 4096
)

type realtimeConfig struct {
	pingInterval time.Duration
	pingTimeout  time.Duration
}

type realtimeOption func(*realtimeConfig)

func defaultRealtimeConfig() realtimeConfig {
	return realtimeConfig{
		pingInterval: defaultWSPingInterval,
		pingTimeout:  defaultWSPingTimeout,
	}
}

func WithPingInterval(interval time.Duration) realtimeOption {
	return func(config *realtimeConfig) {
		if interval > 0 {
			config.pingInterval = interval
		}
	}
}

func WithPingTimeout(timeout time.Duration) realtimeOption {
	return func(config *realtimeConfig) {
		if timeout > 0 {
			config.pingTimeout = timeout
		}
	}
}

type realtimeToken struct {
	user      string
	expiresAt time.Time
}

var wsTokens = struct {
	sync.Mutex
	tokens          map[[sha256.Size]byte]realtimeToken
	lastSweep       time.Time
	sweepTimer      *time.Timer
	sweepGeneration uint64
}{
	tokens: map[[sha256.Size]byte]realtimeToken{},
}

var legacyWSProtocolAuditWarned stdatomic.Bool

func init() {
	database.RegisterResetHook("api.ws_tokens", func() {
		_ = sweepAllWSTokens()
	})
	service.RegisterWSTokenInvalidationHook("api.ws_tokens", sweepAllWSTokens)
}

func (a *ApiService) IssueWSToken(c *gin.Context) {
	if !a.enforceWSHandshakeRateLimit(c, "ws-token") {
		return
	}
	user := GetLoginUser(c)
	if user == "" {
		jsonMsg(c, "wsToken", common.NewError("invalid login"))
		return
	}
	if !a.validateWSOrigin(c, user) {
		return
	}
	now := time.Now()
	expiresAt := now.Add(wsTokenTTL)
	token := common.Random(32)
	wsTokens.Lock()
	maybeSweepWSTokensLocked(now)
	wsTokens.tokens[wsTokenDigest(token)] = realtimeToken{user: user, expiresAt: expiresAt}
	enforceWSTokenCapLocked()
	scheduleWSTokenSweepLocked()
	wsTokens.Unlock()
	jsonObj(c, gin.H{
		"token":     token,
		"expiresAt": expiresAt.Unix(),
	}, nil)
}

func (a *ApiService) RealtimeWS(c *gin.Context) {
	a.realtimeWS(c, defaultRealtimeConfig())
}

func (a *ApiService) RealtimeWSWithOptions(options ...realtimeOption) gin.HandlerFunc {
	config := defaultRealtimeConfig()
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return func(c *gin.Context) {
		a.realtimeWS(c, config)
	}
}

func (a *ApiService) realtimeWS(c *gin.Context, config realtimeConfig) {
	if !a.enforceWSHandshakeRateLimit(c, "ws") {
		return
	}
	user := GetLoginUser(c)
	if !a.validateWSOrigin(c, user) {
		return
	}
	token, legacyProtocol := wsTokenFromRequest(c)
	if legacyProtocol {
		a.recordLegacyWSProtocolAuditOnce(c, user)
	}
	tokenUser, ok := consumeWSToken(token)
	if !ok || tokenUser == "" || tokenUser != user {
		c.Status(http.StatusUnauthorized)
		return
	}
	ip := getRemoteIp(c)
	releaseReservation, ok := realtime.Reserve(user, ip, maxWSPerUser, maxWSPerIP)
	if !ok {
		c.Status(http.StatusTooManyRequests)
		return
	}

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		Subprotocols: []string{wsSubprotocol},
	})
	if err != nil {
		releaseReservation()
		return
	}
	sendCh := make(chan realtime.Event, wsQueueSize)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   user,
		IP:     ip,
		Scope:  realtimeScopeFromContext(c),
		SendCh: sendCh,
		OnDrop: func(reason string) {
			code := wsCloseAuth
			if reason == "slow" {
				code = websocket.StatusPolicyViolation
			}
			_ = conn.Close(code, reason)
		},
	})
	defer func() {
		unregister()
		releaseReservation()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	wsCtx := conn.CloseRead(c.Request.Context())
	heartbeatCtx, stopHeartbeat := context.WithCancel(wsCtx)
	heartbeatDone := startWSHeartbeat(heartbeatCtx, conn, config)
	defer func() {
		stopHeartbeat()
		<-heartbeatDone
	}()

	select {
	case sendCh <- realtime.Event{Type: realtime.Topic("connected"), Ts: time.Now().Unix()}:
	default:
		_ = conn.Close(websocket.StatusPolicyViolation, "slow client")
		return
	}
	for {
		select {
		case event := <-sendCh:
			payload, _ := json.Marshal(event)
			writeCtx, cancel := context.WithTimeout(wsCtx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}
		case <-wsCtx.Done():
			return
		}
	}
}

func realtimeScopeFromContext(c *gin.Context) realtime.Scope {
	switch c.GetString(apiTokenScopeKey) {
	case "":
		return realtime.ScopeAdmin
	case string(realtime.ScopeAdmin):
		return realtime.ScopeAdmin
	case string(realtime.ScopeRead):
		return realtime.ScopeRead
	case string(realtime.ScopeWrite):
		return realtime.ScopeWrite
	case string(realtime.ScopeObservability):
		return realtime.ScopeObservability
	default:
		return realtime.ScopeRead
	}
}

func startWSHeartbeat(ctx context.Context, conn *websocket.Conn, config realtimeConfig) <-chan struct{} {
	done := make(chan struct{})
	pingInterval := config.pingInterval
	pingTimeout := config.pingTimeout
	go func() {
		defer close(done)
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
				err := conn.Ping(pingCtx)
				cancel()
				if err != nil {
					_ = conn.Close(websocket.StatusInternalError, "heartbeat")
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return done
}

func (a *ApiService) enforceWSHandshakeRateLimit(c *gin.Context, endpoint string) bool {
	err := checkWSHandshakeRateLimit(wsHandshakeRateLimitKey(endpoint, getRemoteIp(c)))
	if err == nil {
		return true
	}
	a.recordAudit(c, "", "ws_rate_limited", "realtime", service.AuditSeverityWarn, map[string]any{
		"endpoint": endpoint,
	})
	c.Header("Retry-After", strconv.Itoa(int(wsHandshakeRateLimitWindow/time.Second)))
	if endpoint == "ws-token" {
		c.JSON(http.StatusTooManyRequests, Msg{Success: false, Msg: "wsToken: " + err.Error()})
	} else {
		c.Status(http.StatusTooManyRequests)
	}
	return false
}

func wsTokenFromRequest(c *gin.Context) (string, bool) {
	if token := strings.TrimSpace(c.Query("token")); token != "" {
		return token, false
	}
	var legacy string
	for _, part := range strings.Split(c.GetHeader("Sec-WebSocket-Protocol"), ",") {
		part = strings.TrimSpace(part)
		if token, ok := strings.CutPrefix(part, wsTokenPrefix); ok && token != "" {
			return token, false
		}
		if part != "" && part != wsSubprotocol && legacy == "" {
			legacy = part
		}
	}
	if legacy != "" {
		return legacy, true
	}
	return "", false
}

func (a *ApiService) recordLegacyWSProtocolAuditOnce(c *gin.Context, user string) {
	if !legacyWSProtocolAuditWarned.CompareAndSwap(false, true) {
		return
	}
	a.recordAudit(c, user, "ws_protocol_deprecated", "realtime", service.AuditSeverityWarn, map[string]any{
		"format": "legacy_token_subprotocol",
	})
}

func wsTokenDigest(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}

func (a *ApiService) validateWSOrigin(c *gin.Context, user string) bool {
	originHeader := strings.TrimSpace(c.GetHeader("Origin"))
	if originHeader == "" {
		return true
	}
	webDomain, _ := a.SettingService.GetWebDomain()
	allowed, reason := wsOriginAllowed(originHeader, c.Request.Host, webDomain)
	if allowed {
		return true
	}
	originHost, originScheme := originAuditParts(originHeader)
	a.recordAudit(c, user, "ws_origin_rejected", "realtime", service.AuditSeverityWarn, map[string]any{
		"reason":       reason,
		"originScheme": originScheme,
		"originHost":   originHost,
		"requestHost":  canonicalHostPort(c.Request.Host),
		"webDomain":    canonicalHostname(webDomain),
	})
	c.Status(http.StatusForbidden)
	return false
}

func wsOriginAllowed(originHeader string, requestHost string, webDomain string) (bool, string) {
	originURL, err := url.Parse(originHeader)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		return false, "invalid_origin"
	}
	if originURL.Scheme != "http" && originURL.Scheme != "https" {
		return false, "invalid_scheme"
	}
	if originURL.RawQuery != "" || originURL.Fragment != "" || (originURL.Path != "" && originURL.Path != "/") {
		return false, "invalid_origin"
	}

	originHostPort := canonicalHostPort(originURL.Host)
	if originHostPort == "" {
		return false, "invalid_origin"
	}
	if requestHost != "" && originHostPort == canonicalHostPort(requestHost) {
		return true, "request_host"
	}

	originHost := canonicalHostname(originURL.Host)
	webDomainHost := canonicalHostname(webDomain)
	if webDomainHost != "" && originHost == webDomainHost {
		return true, "web_domain"
	}
	if webDomainHostPort := canonicalHostPort(webDomain); webDomainHostPort != "" && originHostPort == webDomainHostPort {
		return true, "web_domain"
	}
	return false, "host_mismatch"
}

func originAuditParts(originHeader string) (string, string) {
	originURL, err := url.Parse(originHeader)
	if err != nil {
		return "", ""
	}
	return canonicalHostPort(originURL.Host), originURL.Scheme
}

func canonicalHostPort(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = parsed.Host
	}
	if host, port, err := net.SplitHostPort(value); err == nil {
		return strings.TrimSuffix(strings.ToLower(strings.Trim(host, "[]")), ".") + ":" + port
	}
	return strings.TrimSuffix(strings.ToLower(strings.Trim(value, "[]")), ".")
}

func canonicalHostname(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = parsed.Host
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return strings.TrimSuffix(strings.ToLower(strings.Trim(value, "[]")), ".")
}

func consumeWSToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	wsTokens.Lock()
	defer wsTokens.Unlock()

	candidate := wsTokenDigest(token)
	keys := make([][sha256.Size]byte, 0, len(wsTokens.tokens))
	for key := range wsTokens.tokens {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})

	matched := 0
	var matchedKey [sha256.Size]byte
	matchedExpiresAtUnixNano := int64(0)
	matchedUserIndex := 0
	users := make([]string, len(keys))
	for i, key := range keys {
		data := wsTokens.tokens[key]
		users[i] = data.user
		eq := subtle.ConstantTimeCompare(candidate[:], key[:])
		subtle.ConstantTimeCopy(eq, matchedKey[:], key[:])
		matched = subtle.ConstantTimeSelect(eq, 1, matched)
		matchedExpiresAtUnixNano = constantTimeSelectInt64(eq, data.expiresAt.UnixNano(), matchedExpiresAtUnixNano)
		matchedUserIndex = subtle.ConstantTimeSelect(eq, i+1, matchedUserIndex)
	}
	delete(wsTokens.tokens, matchedKey)
	now := time.Now()
	matchedExpiresAt := time.Unix(0, matchedExpiresAtUnixNano)
	if matched != 1 || now.After(matchedExpiresAt) {
		return "", false
	}
	return users[matchedUserIndex-1], true
}

func constantTimeSelectInt64(v int, x int64, y int64) int64 {
	mask := int64(-v)
	return (x & mask) | (y &^ mask)
}

func maybeSweepWSTokensLocked(now time.Time) {
	if wsTokens.lastSweep.IsZero() || now.Sub(wsTokens.lastSweep) > wsTokenSweepInterval {
		sweepWSTokensLocked(now)
	}
}

func runWSTokenSweep(generation uint64) {
	wsTokens.Lock()
	defer wsTokens.Unlock()
	if generation != wsTokens.sweepGeneration {
		return
	}
	wsTokens.sweepTimer = nil
	sweepWSTokensLocked(time.Now())
	scheduleWSTokenSweepLocked()
}

func scheduleWSTokenSweepLocked() {
	if len(wsTokens.tokens) == 0 || wsTokens.sweepTimer != nil {
		return
	}
	generation := wsTokens.sweepGeneration
	wsTokens.sweepTimer = time.AfterFunc(wsTokenSweepInterval, func() {
		runWSTokenSweep(generation)
	})
}

func sweepWSTokensLocked(now time.Time) {
	for token, data := range wsTokens.tokens {
		if now.After(data.expiresAt) {
			delete(wsTokens.tokens, token)
		}
	}
	wsTokens.lastSweep = now
	enforceWSTokenCapLocked()
}

func sweepAllWSTokens() int {
	wsTokens.Lock()
	defer wsTokens.Unlock()
	return sweepAllWSTokensLocked()
}

func sweepAllWSTokensLocked() int {
	count := len(wsTokens.tokens)
	wsTokens.tokens = map[[sha256.Size]byte]realtimeToken{}
	wsTokens.lastSweep = time.Time{}
	if wsTokens.sweepTimer != nil {
		wsTokens.sweepTimer.Stop()
		wsTokens.sweepTimer = nil
	}
	wsTokens.sweepGeneration++
	return count
}

func enforceWSTokenCapLocked() {
	overflow := len(wsTokens.tokens) - maxWSTokens
	if overflow <= 0 {
		return
	}
	entries := make([]struct {
		token     [sha256.Size]byte
		expiresAt time.Time
	}, 0, len(wsTokens.tokens))
	for token, data := range wsTokens.tokens {
		entries = append(entries, struct {
			token     [sha256.Size]byte
			expiresAt time.Time
		}{token: token, expiresAt: data.expiresAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].expiresAt.Equal(entries[j].expiresAt) {
			return bytes.Compare(entries[i].token[:], entries[j].token[:]) < 0
		}
		return entries[i].expiresAt.Before(entries[j].expiresAt)
	})
	for i := 0; i < overflow; i++ {
		delete(wsTokens.tokens, entries[i].token)
	}
}
