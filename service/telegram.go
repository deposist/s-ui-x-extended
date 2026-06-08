package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util"
	"github.com/deposist/s-ui-x/util/common"
	"github.com/deposist/s-ui-x/util/redact"
	"github.com/deposist/s-ui-x/util/ssrf"
	"golang.org/x/net/proxy"
)

type TelegramService struct {
	SettingService
	Runtime *Runtime
}

func (s *TelegramService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

type TelegramResult struct {
	Success    bool          `json:"success"`
	ErrorClass string        `json:"errorClass,omitempty"`
	RetryAfter time.Duration `json:"-"`
}

const (
	telegramQueueCapacity = 256
	telegramMaxRetryAfter = 300 * time.Second
	telegramProxyDialTime = 10 * time.Second
)

var (
	telegramHTTPClientMu sync.RWMutex
	telegramHTTPClient   = &http.Client{Timeout: 10 * time.Second}
	telegramHTTPOverride bool
	telegramHTTPConfig   telegramProxyConfig
)

type telegramProxyConfig struct {
	URL      string
	Username string
	Password string
}

type telegramNotification struct {
	event string
	text  string
}

type telegramNotifier struct {
	capacity int
	send     func(string) TelegramResult
	audit    func(string, map[string]any)
	backoff  []time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once

	mu       sync.Mutex
	cond     *sync.Cond
	queue    []telegramNotification
	done     chan struct{}
	doneOnce sync.Once
	started  bool
	stopped  bool
}

func newTelegramNotifier(capacity int, send func(string) TelegramResult, audit func(string, map[string]any)) *telegramNotifier {
	if capacity <= 0 {
		capacity = telegramQueueCapacity
	}
	notifier := &telegramNotifier{
		capacity: capacity,
		send:     send,
		audit:    audit,
		backoff: []time.Duration{
			500 * time.Millisecond,
			2 * time.Second,
		},
		queue:  make([]telegramNotification, 0, capacity),
		done:   make(chan struct{}),
		stopCh: make(chan struct{}),
	}
	notifier.cond = sync.NewCond(&notifier.mu)
	return notifier
}

func newDefaultTelegramNotifier() *telegramNotifier {
	return newTelegramNotifier(
		telegramQueueCapacity,
		func(text string) TelegramResult {
			return (&TelegramService{}).send(text)
		},
		recordTelegramNotifierAudit,
	)
}

func StopTelegramNotifier(ctx context.Context) error {
	runtime := DefaultRuntime()
	notifier := runtime.telegram()
	if notifier == nil {
		return nil
	}

	err := notifier.Stop(ctx)

	runtime.replaceTelegramNotifierIfCurrent(notifier)
	return err
}

func (n *telegramNotifier) Enqueue(job telegramNotification) {
	n.start()
	if dropped := n.push(job); dropped != nil {
		logger.Warning("telegram notifier queue overflow; dropped event: ", dropped.event)
		n.recordAudit("notifier_overflow", map[string]any{
			"channel":      "telegram",
			"droppedEvent": dropped.event,
			"queuedEvent":  job.event,
		})
	}
}

func (n *telegramNotifier) start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped || n.started {
		return
	}
	n.started = true
	go n.run()
}

func (n *telegramNotifier) push(job telegramNotification) *telegramNotification {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stopped {
		return nil
	}
	if len(n.queue) >= n.capacity {
		dropped := n.queue[0]
		copy(n.queue, n.queue[1:])
		n.queue[len(n.queue)-1] = job
		n.cond.Signal()
		return &dropped
	}
	n.queue = append(n.queue, job)
	n.cond.Signal()
	return nil
}

func (n *telegramNotifier) next() (telegramNotification, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for len(n.queue) == 0 && !n.stopped {
		n.cond.Wait()
	}
	if len(n.queue) == 0 {
		return telegramNotification{}, false
	}
	job := n.queue[0]
	copy(n.queue, n.queue[1:])
	n.queue = n.queue[:len(n.queue)-1]
	return job, true
}

func (n *telegramNotifier) run() {
	defer n.closeDone()
	for {
		job, ok := n.next()
		if !ok {
			return
		}
		n.deliver(job)
	}
}

func (n *telegramNotifier) deliver(job telegramNotification) {
	attempts := len(n.backoff) + 1
	result := TelegramResult{ErrorClass: "unknown"}
	for attempt := 0; attempt < attempts; attempt++ {
		result = n.send(job.text)
		if result.Success {
			return
		}
		if attempt < len(n.backoff) {
			delay := n.backoff[attempt]
			if result.RetryAfter > 0 {
				delay = result.RetryAfter
			}
			if !n.sleepBackoff(delay) {
				return
			}
		}
	}
	if result.ErrorClass == "" {
		result.ErrorClass = "unknown"
	}
	logger.Warning("telegram notification failed: ", result.ErrorClass)
	n.recordAudit("notifier_failed", map[string]any{
		"channel":    "telegram",
		"event":      job.event,
		"errorClass": result.ErrorClass,
		"attempts":   attempts,
	})
}

func (n *telegramNotifier) recordAudit(event string, details map[string]any) {
	if n.audit == nil {
		return
	}
	n.audit(event, details)
}

func (n *telegramNotifier) sleepBackoff(delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-n.stopCh:
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(delay)
	select {
	case <-timer.C:
		return true
	case <-n.stopCh:
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return false
	}
}

func (n *telegramNotifier) closeStopCh() {
	n.stopOnce.Do(func() {
		close(n.stopCh)
	})
}

func (n *telegramNotifier) closeDone() {
	n.doneOnce.Do(func() {
		close(n.done)
	})
}

func (n *telegramNotifier) Stop(ctx context.Context) error {
	n.mu.Lock()
	n.stopped = true
	n.cond.Broadcast()
	started := n.started
	done := n.done
	n.mu.Unlock()
	n.closeStopCh()
	if !started {
		n.closeDone()
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *TelegramService) getTelegramHTTPClient() (*http.Client, error) {
	// A test override always wins (used by the test seam).
	telegramHTTPClientMu.RLock()
	if telegramHTTPOverride {
		client := telegramHTTPClient
		telegramHTTPClientMu.RUnlock()
		return client, nil
	}
	telegramHTTPClientMu.RUnlock()

	// Outbound transport: dial through a running sing-box outbound. Built fresh
	// each call (depends on the live core, which changes across restarts), so it
	// is not cached.
	if mode, _ := s.getString("telegramTransportMode"); mode == "outbound" {
		tag, _ := s.getString("telegramOutboundTag")
		return newCoreOutboundHTTPClient(tag, 10*time.Second)
	}

	cfg, err := s.telegramProxyConfig()
	if err != nil {
		return nil, err
	}
	telegramHTTPClientMu.RLock()
	if telegramHTTPClient != nil && telegramHTTPConfig == cfg {
		client := telegramHTTPClient
		telegramHTTPClientMu.RUnlock()
		return client, nil
	}
	telegramHTTPClientMu.RUnlock()

	telegramHTTPClientMu.Lock()
	defer telegramHTTPClientMu.Unlock()
	if telegramHTTPOverride {
		return telegramHTTPClient, nil
	}
	if telegramHTTPClient != nil && telegramHTTPConfig == cfg {
		return telegramHTTPClient, nil
	}

	client, err := newTelegramHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	telegramHTTPClient = client
	telegramHTTPConfig = cfg
	return client, nil
}

func setTelegramHTTPClient(client *http.Client) func() {
	telegramHTTPClientMu.Lock()
	oldClient := telegramHTTPClient
	oldOverride := telegramHTTPOverride
	oldConfig := telegramHTTPConfig
	telegramHTTPClient = client
	telegramHTTPOverride = true
	telegramHTTPClientMu.Unlock()
	return func() {
		telegramHTTPClientMu.Lock()
		telegramHTTPClient = oldClient
		telegramHTTPOverride = oldOverride
		telegramHTTPConfig = oldConfig
		telegramHTTPClientMu.Unlock()
	}
}

func (s *TelegramService) TestTelegram() TelegramResult {
	return s.send("S-UI Telegram notification test")
}

func EncryptTelegramBackup(plain []byte) ([]byte, []byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	encrypted := make([]byte, 0, len(nonce)+len(plain)+gcm.Overhead())
	encrypted = append(encrypted, nonce...)
	encrypted = gcm.Seal(encrypted, nonce, plain, nil)
	return encrypted, key, nil
}

func (s *TelegramService) SendTelegramDocument(filename string, data []byte, caption string) TelegramResult {
	enabled, err := s.telegramEnabled()
	if err != nil {
		return TelegramResult{ErrorClass: "settings"}
	}
	if !enabled {
		return TelegramResult{ErrorClass: "disabled"}
	}
	token, err := s.getString("telegramBotToken")
	if err != nil || token == "" {
		return TelegramResult{ErrorClass: "missing_token"}
	}
	chatID, err := s.getString("telegramChatID")
	if err != nil || chatID == "" {
		return TelegramResult{ErrorClass: "missing_chat"}
	}

	bodyReader, bodyWriter := io.Pipe()
	writer := multipart.NewWriter(bodyWriter)
	writeErr := make(chan error, 1)
	go func() {
		err := writeTelegramDocumentMultipart(writer, chatID, filename, data, caption)
		if err == nil {
			err = writer.Close()
		}
		if err != nil {
			_ = bodyWriter.CloseWithError(err)
			writeErr <- err
			return
		}
		writeErr <- bodyWriter.Close()
	}()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.telegram.org/bot"+token+"/sendDocument", bodyReader)
	if err != nil {
		_ = bodyReader.CloseWithError(err)
		<-writeErr
		return TelegramResult{ErrorClass: "request"}
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client, err := s.getTelegramHTTPClient()
	if err != nil {
		_ = bodyReader.CloseWithError(err)
		<-writeErr
		return TelegramResult{ErrorClass: "proxy"}
	}
	resp, err := client.Do(req)
	if err != nil {
		_ = bodyReader.CloseWithError(err)
		<-writeErr
		return TelegramResult{ErrorClass: "network"}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if err := <-writeErr; err != nil {
		return TelegramResult{ErrorClass: "payload"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TelegramResult{ErrorClass: telegramStatusErrorClass(resp.StatusCode)}
	}
	return TelegramResult{Success: true}
}

func writeTelegramDocumentMultipart(writer *multipart.Writer, chatID string, filename string, data []byte, caption string) error {
	if err := writer.WriteField("chat_id", chatID); err != nil {
		return err
	}
	if caption = telegramCaption(caption); caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return err
		}
	}
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, bytes.NewReader(data))
	return err
}

func (s *TelegramService) NotifyTelegramEvent(event string, fields map[string]string) {
	enabled, err := s.telegramEnabled()
	if err != nil || !enabled {
		return
	}
	msg := "S-UI event: " + redact.String(event)
	for key, value := range fields {
		if value == "" {
			continue
		}
		if redact.IsSensitiveKey(key) {
			value = redact.Marker
		} else {
			value = redact.String(value)
		}
		msg += "\n" + key + ": " + value
	}
	notifier := s.runtime().telegram()
	if notifier != nil {
		notifier.Enqueue(telegramNotification{event: event, text: msg})
	}
}

func (s *TelegramService) send(text string) TelegramResult {
	enabled, err := s.telegramEnabled()
	if err != nil {
		return TelegramResult{ErrorClass: "settings"}
	}
	if !enabled {
		return TelegramResult{ErrorClass: "disabled"}
	}
	token, err := s.getString("telegramBotToken")
	if err != nil || token == "" {
		return TelegramResult{ErrorClass: "missing_token"}
	}
	chatID, err := s.getString("telegramChatID")
	if err != nil || chatID == "" {
		return TelegramResult{ErrorClass: "missing_chat"}
	}
	payload, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    redact.String(text),
	})
	if err != nil {
		return TelegramResult{ErrorClass: "payload"}
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.telegram.org/bot"+token+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return TelegramResult{ErrorClass: "request"}
	}
	req.Header.Set("Content-Type", "application/json")
	client, err := s.getTelegramHTTPClient()
	if err != nil {
		return TelegramResult{ErrorClass: "proxy"}
	}
	resp, err := client.Do(req)
	if err != nil {
		return TelegramResult{ErrorClass: "network"}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return TelegramResult{
			ErrorClass: telegramStatusErrorClass(resp.StatusCode),
			RetryAfter: telegramRetryAfter(resp.StatusCode, body),
		}
	}
	return TelegramResult{Success: true}
}

func telegramRetryAfter(status int, body []byte) time.Duration {
	if status != http.StatusTooManyRequests || len(body) == 0 {
		return 0
	}
	var response struct {
		OK         bool `json:"ok"`
		ErrorCode  int  `json:"error_code"`
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return 0
	}
	if response.ErrorCode != http.StatusTooManyRequests || response.Parameters.RetryAfter <= 0 {
		return 0
	}
	retryAfter := time.Duration(response.Parameters.RetryAfter) * time.Second
	if retryAfter > telegramMaxRetryAfter {
		return telegramMaxRetryAfter
	}
	return retryAfter
}

func telegramStatusErrorClass(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusNotFound:
		return "chat_not_found"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "unknown"
	}
}

func telegramCaption(caption string) string {
	return util.SafeHeader(redact.String(caption), 1024)
}

func (s *TelegramService) telegramEnabled() (bool, error) {
	return s.getBool("telegramEnabled")
}

func (s *TelegramService) telegramProxyConfig() (telegramProxyConfig, error) {
	proxyURL, err := s.getString("telegramProxyURL")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	username, err := s.getString("telegramProxyUsername")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	password, err := s.getString("telegramProxyPassword")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	return telegramProxyConfig{
		URL:      proxyURL,
		Username: username,
		Password: password,
	}, nil
}

func newTelegramHTTPClient(cfg telegramProxyConfig) (*http.Client, error) {
	if cfg.URL == "" {
		return &http.Client{Timeout: 10 * time.Second}, nil
	}
	if err := validateTelegramProxyURL(cfg.URL); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	if cfg.Username != "" || cfg.Password != "" {
		parsed.User = url.UserPassword(cfg.Username, cfg.Password)
	}
	switch parsed.Scheme {
	case "http", "https":
		return &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsed),
			},
		}, nil
	case "socks5":
		var auth *proxy.Auth
		username := cfg.Username
		password := cfg.Password
		if parsed.User != nil && username == "" && password == "" {
			username = parsed.User.Username()
			password, _ = parsed.User.Password()
		}
		if username != "" || password != "" {
			auth = &proxy.Auth{User: username, Password: password}
		}
		transport, err := newTelegramSOCKS5Transport(parsed.Host, auth)
		if err != nil {
			return nil, err
		}
		return &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		}, nil
	default:
		return nil, common.NewError("unsupported telegram proxy scheme")
	}
}

func newTelegramSOCKS5Transport(proxyHost string, auth *proxy.Auth) (*http.Transport, error) {
	forward := &net.Dialer{Timeout: telegramProxyDialTime}
	dialer, err := proxy.SOCKS5("tcp", proxyHost, auth, forward)
	if err != nil {
		return nil, err
	}
	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, common.NewError("telegram socks5 proxy does not support context dial")
	}
	return &http.Transport{
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			dialCtx, cancel := context.WithTimeout(ctx, telegramProxyDialTime)
			defer cancel()
			return contextDialer.DialContext(dialCtx, network, address)
		},
	}, nil
}

func validateTelegramProxyURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	if parsed, err := url.Parse(rawURL); err == nil && parsed.User != nil {
		return common.NewError("proxy url must not contain credentials; use the username/password fields")
	}
	return ssrf.ValidateOutboundURL(context.Background(), rawURL, "http", "https", "socks5")
}

func recordTelegramNotifierAudit(event string, details map[string]any) {
	if database.GetDB() == nil {
		return
	}
	if err := (&AuditService{}).Record(AuditEvent{
		Event:    event,
		Resource: "notifier",
		Severity: AuditSeverityWarn,
		Details:  details,
	}); err != nil {
		logger.Warning("telegram notifier audit failed:", err)
	}
}
