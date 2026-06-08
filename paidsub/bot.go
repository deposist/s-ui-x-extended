package paidsub

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
)

// Bot is the long-poll receiver for the client-facing Telegram bot. One Bot
// instance is the sole getUpdates consumer for its token.
type Bot struct {
	setting      service.SettingService
	svc          *PaidSubService
	payments     *PaymentService
	client       *http.Client
	token        string
	cmdLimiter   *rateLimiter
	startLimiter *rateLimiter
}

func newBot() *Bot {
	return &Bot{
		svc:          NewService(),
		payments:     NewPaymentService(),
		cmdLimiter:   newRateLimiter(20, 60),
		startLimiter: newRateLimiter(0, 60), // cap supplied per-call from settings
	}
}

// ---- lifecycle (package singleton) ----

var (
	botMu     sync.Mutex
	botCancel context.CancelFunc
	botDone   chan struct{}
)

// StartBot launches the receiver goroutine if not already running. Idempotent.
func StartBot() {
	botMu.Lock()
	defer botMu.Unlock()
	if botCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	botCancel = cancel
	botDone = done
	b := newBot()
	go b.run(ctx, done)
}

// StopBot signals the receiver to stop and waits up to ctx for it to finish.
func StopBot(ctx context.Context) error {
	botMu.Lock()
	cancel := botCancel
	done := botDone
	botCancel = nil
	botDone = nil
	botMu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// newSenderBot builds a Bot ready to SEND (not poll) — used by the payment poll
// job to notify users out-of-band. Returns an error if the bot token is unset.
func newSenderBot() (*Bot, error) {
	b := newBot()
	token, err := b.setting.GetPaidSubBotToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("paidsub: bot token not configured")
	}
	poll, _ := b.setting.GetPaidSubBotPollSeconds()
	client, err := service.NewPaidSubHTTPClient(time.Duration(poll+10) * time.Second)
	if err != nil {
		return nil, err
	}
	b.client = client
	b.token = token
	return b, nil
}

// sleepCtx sleeps for d or until ctx is cancelled. Returns true if cancelled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-t.C:
		return false
	}
}

func (b *Bot) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	backoff := time.Second
	const maxBackoff = 60 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		enabled, err := b.setting.GetPaidSubEnabled()
		if err != nil || !enabled {
			if sleepCtx(ctx, 5*time.Second) {
				return
			}
			continue
		}
		token, err := b.setting.GetPaidSubBotToken()
		if err != nil || token == "" {
			if sleepCtx(ctx, 5*time.Second) {
				return
			}
			continue
		}
		poll, _ := b.setting.GetPaidSubBotPollSeconds()
		client, err := service.NewPaidSubHTTPClient(time.Duration(poll+10) * time.Second)
		if err != nil {
			logger.Warning("paidsub: build http client: ", err)
			if sleepCtx(ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff, maxBackoff)
			continue
		}
		// Close the previous client's idle keep-alive connections before
		// replacing it; a discarded *http.Transport (proxy/outbound mode) does
		// not auto-close them, so rebuilding every loop would leak sockets.
		if b.client != nil && b.client != client {
			b.client.CloseIdleConnections()
		}
		b.client = client
		b.token = token

		offset, _ := b.setting.GetPaidSubUpdateOffset()
		updates, err := b.getUpdates(ctx, offset, poll)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			wait := b.classifyError(err, backoff)
			if sleepCtx(ctx, wait) {
				return
			}
			backoff = nextBackoff(backoff, maxBackoff)
			continue
		}
		backoff = time.Second

		maxID := offset
		for i := range updates {
			b.handleUpdate(ctx, &updates[i])
			if updates[i].UpdateID >= maxID {
				maxID = updates[i].UpdateID + 1
			}
		}
		if maxID != offset {
			if err := b.setting.SetPaidSubUpdateOffset(maxID); err != nil {
				logger.Warning("paidsub: persist offset: ", err)
			}
		}
	}
}

func nextBackoff(cur, max time.Duration) time.Duration {
	cur *= 2
	if cur > max {
		return max
	}
	return cur
}

// classifyError returns how long to wait after a getUpdates failure, handling
// 409 (a second consumer / webhook set) and 401 (revoked token) specially. It
// never logs the token (tgAPIError carries only code+description).
func (b *Bot) classifyError(err error, backoff time.Duration) time.Duration {
	var apiErr *tgAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case http.StatusConflict: // 409: another getUpdates consumer or webhook
			logger.Warning("paidsub: getUpdates conflict (409); another consumer or webhook is active")
			return 30 * time.Second
		case http.StatusUnauthorized: // 401: token revoked/invalid
			logger.Warning("paidsub: bot token unauthorized (401); pausing until settings change")
			return 60 * time.Second
		case http.StatusTooManyRequests:
			if apiErr.RetryAfter > 0 {
				return time.Duration(apiErr.RetryAfter) * time.Second
			}
		}
		logger.Warning("paidsub: getUpdates error: ", apiErr.Error())
		return backoff
	}
	logger.Warning("paidsub: getUpdates failed")
	return backoff
}

// ---- dispatch ----

func (b *Bot) handleUpdate(ctx context.Context, u *tgUpdate) {
	switch {
	case u.PreCheckoutQuery != nil:
		b.handlePreCheckout(ctx, u.PreCheckoutQuery)
	case u.Message != nil && u.Message.SuccessfulPayment != nil:
		b.handleSuccessfulPayment(ctx, u.Message)
	case u.Message != nil:
		b.handleMessage(ctx, u.Message)
	case u.CallbackQuery != nil:
		b.handleCallback(ctx, u.CallbackQuery)
	}
}

func (b *Bot) handleMessage(ctx context.Context, m *tgMessage) {
	if m.From == nil || m.From.ID <= 0 || m.From.IsBot {
		return
	}
	if m.Chat.Type != "private" {
		return
	}
	l := pickLang(m.From.LanguageCode)
	if !b.cmdLimiter.allow(m.From.ID, nowUnix()) {
		return // silent drop
	}
	cmd, _ := parseCommand(m.Text)
	switch cmd {
	case "/help":
		_ = b.sendMessage(ctx, m.Chat.ID, tr(l, "help"), nil)
	case "/links", "/sub":
		b.cmdLinks(ctx, m.Chat.ID, m.From.ID, l)
	case "/qr":
		b.cmdQR(ctx, m.Chat.ID, m.From.ID, l)
	case "/stats", "/usage":
		b.cmdStats(ctx, m.Chat.ID, m.From.ID, l)
	default: // /start, unknown, or plain text → open menu
		b.cmdStart(ctx, m.Chat.ID, m.From, l)
	}
}

func (b *Bot) handleCallback(ctx context.Context, cq *tgCallbackQuery) {
	if cq.From.ID <= 0 || cq.From.IsBot {
		return
	}
	l := pickLang(cq.From.LanguageCode)
	if !b.cmdLimiter.allow(cq.From.ID, nowUnix()) {
		_ = b.answerCallback(ctx, cq.ID, tr(l, "rate_limited"))
		return
	}
	_ = b.answerCallback(ctx, cq.ID, "")
	var chatID int64
	if cq.Message != nil {
		chatID = cq.Message.Chat.ID
	}
	if chatID == 0 {
		return
	}
	data := cq.Data
	switch {
	case data == "links":
		b.cmdLinks(ctx, chatID, cq.From.ID, l)
	case data == "qr":
		b.cmdQR(ctx, chatID, cq.From.ID, l)
	case data == "stats":
		b.cmdStats(ctx, chatID, cq.From.ID, l)
	case data == "help":
		_ = b.sendMessage(ctx, chatID, tr(l, "help"), nil)
	case data == "menu":
		b.cmdStart(ctx, chatID, &cq.From, l)
	case data == "payment":
		b.cmdPaymentMenu(ctx, chatID, l)
	case data == "orders":
		b.cmdMyOrders(ctx, chatID, cq.From.ID, l)
	case data == "refund":
		b.cmdRefundMenu(ctx, chatID, cq.From.ID, l)
	case strings.HasPrefix(data, "refund:"):
		if id, ok := parseUintArg(data, "refund:"); ok {
			b.handleRefundRequest(ctx, chatID, cq.From.ID, id, l)
		}
	case data == "buy":
		b.cmdBuy(ctx, chatID, cq.From.ID, l)
	case strings.HasPrefix(data, "tariff:"):
		if id, ok := parseUintArg(data, "tariff:"); ok {
			b.handleTariffSelect(ctx, chatID, cq.From.ID, id, l)
		}
	case strings.HasPrefix(data, "pay:"):
		if tid, kind, ok := parsePayData(data); ok {
			b.handlePay(ctx, chatID, cq.From.ID, tid, kind, l)
		}
	case strings.HasPrefix(data, "paid:"):
		if id, ok := parseUintArg(data, "paid:"); ok {
			b.handleManualPaid(ctx, chatID, cq.From.ID, id, l)
		}
	}
}

func parseUintArg(data, prefix string) (uint, bool) {
	v, err := strconv.ParseUint(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}

func parsePayData(data string) (uint, string, bool) {
	parts := strings.Split(strings.TrimPrefix(data, "pay:"), ":")
	if len(parts) != 2 {
		return 0, "", false
	}
	v, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || v == 0 || parts[1] == "" {
		return 0, "", false
	}
	return uint(v), parts[1], true
}

// handlePreCheckout / handleSuccessfulPayment are implemented in payment.go
// (Phase 5). Declared here as no-ops would shadow them; the real methods live
// in payment.go.

// ---- commands ----

func (b *Bot) cmdStart(ctx context.Context, chatID int64, from *tgUser, l lang) {
	_, err := b.svc.ClientByTgUserId(from.ID)
	if err != nil {
		// Only a genuine "not found" may lead to auto-registration. A transient
		// DB error must NOT be treated as unbound (that would auto-create and
		// rebind a new client, orphaning an existing subscription).
		if !database.IsNotFound(err) {
			logger.Warning("paidsub: client lookup failed: ", err)
			_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
			return
		}
		if b.tryAutoRegister(ctx, chatID, from, l) {
			return
		}
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	greeting := tr(l, "greeting")
	if custom, _ := b.setting.GetPaidSubGreeting(); strings.TrimSpace(custom) != "" {
		greeting = truncateRunes(custom, 4096)
	}
	_ = b.sendMessage(ctx, chatID, greeting, b.menuKeyboard(l))
}

func (b *Bot) cmdLinks(ctx context.Context, chatID int64, tgID int64, l lang) {
	client, err := b.svc.ClientByTgUserId(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	text := b.buildLinksText(client, l)
	for _, chunk := range chunkText(text, 4000) {
		_ = b.sendMessage(ctx, chatID, chunk, nil)
	}
}

func (b *Bot) cmdQR(ctx context.Context, chatID int64, tgID int64, l lang) {
	client, err := b.svc.ClientByTgUserId(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	sub, err := b.subURL(client)
	if err != nil || sub == "" {
		_ = b.sendMessage(ctx, chatID, tr(l, "links_none"), nil)
		return
	}
	png, err := renderQR(sub)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	if err := b.sendPhoto(ctx, chatID, png, tr(l, "qr_caption_sub")); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
	}
}

func (b *Bot) cmdStats(ctx context.Context, chatID int64, tgID int64, l lang) {
	client, err := b.svc.ClientByTgUserId(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	_ = b.sendMessage(ctx, chatID, b.buildStatsText(client, l), b.menuKeyboard(l))
}

// cmdPaymentMenu opens the "Payment" submenu (buy/renew, my purchases, refund).
func (b *Bot) cmdPaymentMenu(ctx context.Context, chatID int64, l lang) {
	_ = b.sendMessage(ctx, chatID, tr(l, "payment_title"), b.paymentMenuKeyboard(l))
}

// cmdMyOrders lists the requesting user's own orders (read-only history).
func (b *Bot) cmdMyOrders(ctx context.Context, chatID int64, tgID int64, l lang) {
	if _, err := b.svc.ClientByTgUserId(tgID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	orders, err := b.payments.OrdersForTgUser(tgID, 20)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	if len(orders) == 0 {
		_ = b.sendMessage(ctx, chatID, tr(l, "orders_none"), b.backToPaymentKeyboard(l))
		return
	}
	chunks := chunkText(b.buildOrdersText(orders, l), 4000)
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			_ = b.sendMessage(ctx, chatID, chunk, b.backToPaymentKeyboard(l))
		} else {
			_ = b.sendMessage(ctx, chatID, chunk, nil)
		}
	}
}

// cmdRefundMenu lists the user's refundable (paid) orders as tappable buttons.
func (b *Bot) cmdRefundMenu(ctx context.Context, chatID int64, tgID int64, l lang) {
	if _, err := b.svc.ClientByTgUserId(tgID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	orders, err := b.payments.RefundableOrdersForTgUser(tgID, 20)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	if len(orders) == 0 {
		_ = b.sendMessage(ctx, chatID, tr(l, "refund_none"), b.backToPaymentKeyboard(l))
		return
	}
	names := b.tariffNameMap()
	var rows [][]inlineButton
	for i := range orders {
		o := orders[i]
		rows = append(rows, []inlineButton{{Text: refundOrderButtonLabel(&o, names), CallbackData: fmt.Sprintf("refund:%d", o.Id)}})
	}
	rows = append(rows, []inlineButton{{Text: tr(l, "menu_back"), CallbackData: "payment"}})
	_ = b.sendMessage(ctx, chatID, tr(l, "refund_choose"), &inlineKeyboard{InlineKeyboard: rows})
}

// handleRefundRequest processes a refund button. Stars are refunded
// programmatically (claim-and-rollback FIRST, then return the money); every
// other provider cannot be refunded via the Bot API, so it sends the admin a
// request. It never acts on another user's order.
func (b *Bot) handleRefundRequest(ctx context.Context, chatID int64, tgID int64, orderID uint, l lang) {
	order, err := b.payments.getOrder(orderID)
	if err != nil || order.TelegramUserId != tgID {
		return
	}
	if order.Status != StatusPaid {
		_ = b.sendMessage(ctx, chatID, tr(l, "refund_not_eligible"), b.backToPaymentKeyboard(l))
		return
	}
	if order.Provider != string(ProviderStars) {
		(&service.TelegramService{}).NotifyTelegramEvent("paidsub_refund_request", map[string]string{
			"orderId":  fmt.Sprintf("%d", order.Id),
			"clientId": fmt.Sprintf("%d", order.ClientId),
			"provider": order.Provider,
		})
		_ = b.sendMessage(ctx, chatID, tr(l, "refund_requested"), b.backToPaymentKeyboard(l))
		return
	}
	// Stars: the admin policy (paidSubRefundRevoke) decides rollback; the user
	// does not choose, to prevent buy → refund → keep-using abuse.
	revoke, _ := b.setting.GetPaidSubRefundRevoke()
	if err := b.payments.finalizeRefund(order.Id, revoke); err != nil {
		if errors.Is(err, errAlreadyApplied) {
			_ = b.sendMessage(ctx, chatID, tr(l, "refund_done"), b.backToPaymentKeyboard(l))
			return
		}
		logger.Warning("paidsub: finalize refund failed: ", err)
		_ = b.sendMessage(ctx, chatID, tr(l, "error"), nil)
		return
	}
	charge := strings.TrimPrefix(order.ProviderChargeID, "tg:")
	// An "already refunded" response means a concurrent refund (e.g. the admin
	// panel) returned the money first — treat it as success, not a failure.
	if rerr := b.refundStarPayment(ctx, order.TelegramUserId, charge); rerr != nil && !isAlreadyRefunded(rerr) {
		logger.Warning("paidsub: refundStarPayment failed; manual refund needed")
		(&service.TelegramService{}).NotifyTelegramEvent("paidsub_refund_failed", map[string]string{
			"orderId":  fmt.Sprintf("%d", order.Id),
			"clientId": fmt.Sprintf("%d", order.ClientId),
		})
		_ = b.sendMessage(ctx, chatID, tr(l, "refund_requested"), b.backToPaymentKeyboard(l))
		return
	}
	(&service.TelegramService{}).NotifyTelegramEvent("paidsub_refunded", map[string]string{
		"orderId":  fmt.Sprintf("%d", order.Id),
		"clientId": fmt.Sprintf("%d", order.ClientId),
	})
	_ = b.sendMessage(ctx, chatID, tr(l, "refund_done"), b.backToPaymentKeyboard(l))
}

// ---- content builders ----

func (b *Bot) menuKeyboard(l lang) *inlineKeyboard {
	return &inlineKeyboard{InlineKeyboard: [][]inlineButton{
		{{Text: tr(l, "menu_links"), CallbackData: "links"}, {Text: tr(l, "menu_qr"), CallbackData: "qr"}},
		{{Text: tr(l, "menu_stats"), CallbackData: "stats"}, {Text: tr(l, "menu_payment"), CallbackData: "payment"}},
		{{Text: tr(l, "menu_help"), CallbackData: "help"}},
	}}
}

// paymentMenuKeyboard is the "Payment" submenu: buy/renew, my purchases, refund.
func (b *Bot) paymentMenuKeyboard(l lang) *inlineKeyboard {
	return &inlineKeyboard{InlineKeyboard: [][]inlineButton{
		{{Text: tr(l, "menu_buy"), CallbackData: "buy"}},
		{{Text: tr(l, "menu_orders"), CallbackData: "orders"}},
		{{Text: tr(l, "menu_refund"), CallbackData: "refund"}},
		{{Text: tr(l, "menu_back"), CallbackData: "menu"}},
	}}
}

// backToPaymentKeyboard is a single "Back" button returning to the payment menu.
func (b *Bot) backToPaymentKeyboard(l lang) *inlineKeyboard {
	return &inlineKeyboard{InlineKeyboard: [][]inlineButton{
		{{Text: tr(l, "menu_back"), CallbackData: "payment"}},
	}}
}

func (b *Bot) subURL(client *model.Client) (string, error) {
	if client.SubSecret == "" {
		return "", nil
	}
	host, _ := b.setting.GetWebDomain()
	base, err := b.setting.GetFinalSubURI(host)
	if err != nil {
		return "", err
	}
	if base == "" {
		return "", nil
	}
	return base + client.SubSecret, nil
}

func (b *Bot) buildLinksText(client *model.Client, l lang) string {
	var sb strings.Builder
	sb.WriteString(tr(l, "links_title") + "\n")
	if sub, err := b.subURL(client); err == nil && sub != "" {
		sb.WriteString(sub + "\n")
	}
	if len(client.Links) > 0 {
		links := b.svc.Link.GetLinks(&client.Links, "all", "")
		if len(links) > 0 {
			sb.WriteString("\n")
			for _, lk := range links {
				sb.WriteString(lk + "\n")
			}
		}
	}
	out := strings.TrimSpace(sb.String())
	if out == tr(l, "links_title") || out == "" {
		return tr(l, "links_none")
	}
	return out
}

func (b *Bot) buildStatsText(client *model.Client, l lang) string {
	used := client.Up + client.Down
	var sb strings.Builder
	sb.WriteString(tr(l, "stats_title") + "\n\n")
	sb.WriteString(fmt.Sprintf("%s: %s\n", tr(l, "stats_used"), humanBytes(used)))
	if client.Volume > 0 {
		pct := int(used * 100 / client.Volume)
		if pct > 100 {
			pct = 100
		}
		sb.WriteString(fmt.Sprintf("%s: %s (%d%%)\n", tr(l, "stats_limit"), humanBytes(client.Volume), pct))
		sb.WriteString(progressBar(pct) + "\n")
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", tr(l, "stats_limit"), tr(l, "stats_unlim")))
	}
	if client.Expiry > 0 {
		if client.Expiry < nowUnix() {
			sb.WriteString(fmt.Sprintf("%s: %s\n", tr(l, "stats_expiry"), tr(l, "stats_expired")))
		} else {
			days := (client.Expiry - nowUnix()) / 86400
			sb.WriteString(fmt.Sprintf("%s: %d %s\n", tr(l, "stats_expiry"), days, tr(l, "stats_days")))
		}
	}
	if client.Enable {
		sb.WriteString("✅ " + tr(l, "stats_enabled") + "\n")
	} else {
		sb.WriteString("⛔ " + tr(l, "stats_disabled") + "\n")
	}
	if b.isOnline(client.Name) {
		sb.WriteString("🟢 " + tr(l, "stats_online") + "\n")
	} else {
		sb.WriteString("⚪ " + tr(l, "stats_offline") + "\n")
	}
	return strings.TrimSpace(sb.String())
}

func (b *Bot) isOnline(name string) bool {
	onl, err := b.svc.Stats.GetOnlines()
	if err != nil {
		return false
	}
	for _, n := range onl.User {
		if n == name {
			return true
		}
	}
	return false
}

// tariffNameMap returns tariffId → name for labelling orders (best-effort).
func (b *Bot) tariffNameMap() map[uint]string {
	names := map[uint]string{}
	all, err := b.payments.tariffs.GetAll()
	if err != nil {
		return names
	}
	for i := range all {
		names[all[i].Id] = all[i].Name
	}
	return names
}

func (b *Bot) buildOrdersText(orders []PaymentOrder, l lang) string {
	names := b.tariffNameMap()
	var sb strings.Builder
	sb.WriteString(tr(l, "orders_title") + "\n")
	for i := range orders {
		o := orders[i]
		date := ""
		if o.CreatedAt > 0 {
			date = time.Unix(o.CreatedAt, 0).Format("2006-01-02")
		}
		sb.WriteString(fmt.Sprintf("\n• %s — %s\n  %s · %s",
			orderTariffName(&o, names),
			formatOrderAmount(o.Amount, o.Currency),
			orderStatusLabel(o.Status, l),
			date,
		))
	}
	return strings.TrimSpace(sb.String())
}

func orderTariffName(o *PaymentOrder, names map[uint]string) string {
	if name := names[o.TariffId]; name != "" {
		return name
	}
	return "#" + strconv.FormatUint(uint64(o.Id), 10)
}

func refundOrderButtonLabel(o *PaymentOrder, names map[uint]string) string {
	return fmt.Sprintf("%s — %s", orderTariffName(o, names), formatOrderAmount(o.Amount, o.Currency))
}

// orderStatusLabel localizes a status, falling back to the raw value.
func orderStatusLabel(status string, l lang) string {
	key := "order_status_" + status
	if v := tr(l, key); v != key {
		return v
	}
	return status
}

// ---- helpers ----

func parseCommand(text string) (cmd string, arg string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", ""
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}
	cmd = fields[0]
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i]
	}
	if len(fields) > 1 {
		arg = fields[1]
	}
	return strings.ToLower(cmd), arg
}

func humanBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func progressBar(pct int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct / 10
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}

func chunkText(s string, max int) []string {
	if len(s) <= max {
		return []string{s}
	}
	var chunks []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			chunks = append(chunks, strings.TrimRight(cur.String(), "\n"))
			cur.Reset()
		}
	}
	for _, line := range strings.Split(s, "\n") {
		// Hard-split a single line longer than max (rune-safe), so no emitted
		// chunk can exceed the limit.
		for len([]rune(line)) > max {
			flush()
			r := []rune(line)
			chunks = append(chunks, string(r[:max]))
			line = string(r[max:])
		}
		if cur.Len()+len(line)+1 > max && cur.Len() > 0 {
			flush()
		}
		cur.WriteString(line + "\n")
	}
	flush()
	return chunks
}
