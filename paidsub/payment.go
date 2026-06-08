package paidsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"

	"gorm.io/gorm"
)

var errAlreadyApplied = errors.New("order already finalized")
var errRefundNotApplicable = errors.New("order is not refundable")

// isAlreadyRefunded reports whether a refundStarPayment error means the charge
// was already refunded (e.g. by a concurrent refund via the other path).
// Telegram is idempotent at the charge level, so this is a success — not a
// failure — and must not be reported to the admin/user as "refund failed".
func isAlreadyRefunded(err error) bool {
	var apiErr *tgAPIError
	if errors.As(err, &apiErr) {
		return strings.Contains(strings.ToUpper(apiErr.Description), "ALREADY_REFUNDED")
	}
	return false
}

// PaymentService orchestrates orders, invoices and renewals. Logic is scoped to
// the resolved client; amounts are snapshotted server-side from the tariff.
type PaymentService struct {
	setting service.SettingService
	tariffs TariffService
}

func NewPaymentService() *PaymentService { return &PaymentService{} }

// providerByKind builds a configured provider if it is enabled and has its
// token set; otherwise nil.
func (p *PaymentService) providerByKind(kind ProviderKind) PaymentProvider {
	s := &p.setting
	switch kind {
	case ProviderStars:
		if on, _ := s.GetPaidSubStarsEnabled(); on {
			return &telegramProvider{kind: ProviderStars}
		}
	case ProviderYooKassa:
		if on, _ := s.GetPaidSubYooKassaEnabled(); on {
			if tok, _ := s.GetPaidSubYooKassaToken(); tok != "" {
				return &telegramProvider{kind: ProviderYooKassa, token: tok}
			}
		}
	case ProviderStripe:
		if on, _ := s.GetPaidSubStripeEnabled(); on {
			if tok, _ := s.GetPaidSubStripeToken(); tok != "" {
				return &telegramProvider{kind: ProviderStripe, token: tok}
			}
		}
	case ProviderPayMaster:
		if on, _ := s.GetPaidSubPayMasterEnabled(); on {
			if tok, _ := s.GetPaidSubPayMasterToken(); tok != "" {
				return &telegramProvider{kind: ProviderPayMaster, token: tok}
			}
		}
	case ProviderCryptoBot:
		if on, _ := s.GetPaidSubCryptoBotEnabled(); on {
			if tok, _ := s.GetPaidSubCryptoBotToken(); tok != "" {
				return &cryptoBotProvider{token: tok}
			}
		}
	case ProviderExternal:
		if on, _ := s.GetPaidSubExternalEnabled(); on {
			if tmpl, _ := s.GetPaidSubExternalUrlTemplate(); tmpl != "" {
				return &externalProvider{template: tmpl}
			}
		}
	}
	return nil
}

// enabledProvidersForTariff returns providers usable for a tariff: Stars needs
// StarsAmount>0, fiat providers need Price>0. Zero-price tariffs are not
// purchasable (anti free-renewal).
func (p *PaymentService) enabledProvidersForTariff(t *Tariff) []PaymentProvider {
	var kinds []ProviderKind
	if t.StarsAmount > 0 {
		kinds = append(kinds, ProviderStars)
	}
	if t.Price > 0 {
		kinds = append(kinds, ProviderYooKassa, ProviderStripe, ProviderPayMaster, ProviderCryptoBot, ProviderExternal)
	}
	var out []PaymentProvider
	for _, k := range kinds {
		if prov := p.providerByKind(k); prov != nil {
			out = append(out, prov)
		}
	}
	return out
}

// CreateOrder snapshots the price from the tariff, persists a pending order, and
// asks the provider to prepare an invoice.
func (p *PaymentService) CreateOrder(ctx context.Context, client *model.Client, tariff *Tariff, kind ProviderKind, tgUserId int64) (*PaymentOrder, *Invoice, error) {
	prov := p.providerByKind(kind)
	if prov == nil {
		return nil, nil, fmt.Errorf("provider not available")
	}
	var amount int64
	var currency string
	if kind == ProviderStars {
		if tariff.StarsAmount <= 0 {
			return nil, nil, fmt.Errorf("tariff has no stars price")
		}
		amount = tariff.StarsAmount
		currency = "XTR"
	} else {
		if tariff.Price <= 0 {
			return nil, nil, fmt.Errorf("tariff has no price")
		}
		amount = tariff.Price
		currency = tariff.Currency
	}
	ttlMin, _ := p.setting.GetPaidSubOrderTTLMinutes()
	now := nowUnix()
	order := &PaymentOrder{
		ClientId:       client.Id,
		TariffId:       tariff.Id,
		Provider:       string(kind),
		Amount:         amount,
		Currency:       currency,
		Status:         StatusPending,
		TelegramUserId: tgUserId,
		IdempotencyKey: common.Random(32),
		CreatedAt:      now,
		ExpiresAt:      now + int64(ttlMin)*60,
	}
	db := database.GetDB()
	if err := db.Create(order).Error; err != nil {
		return nil, nil, err
	}
	inv, err := prov.CreateInvoice(ctx, order, tariff, client)
	if err != nil {
		return nil, nil, err
	}
	upd := map[string]any{}
	if inv.PayURL != "" {
		upd["external_url"] = inv.PayURL
	}
	if inv.ProviderRef != "" {
		ref, _ := json.Marshal(map[string]string{"ref": inv.ProviderRef})
		upd["provider_payload"] = ref
	}
	if len(upd) > 0 {
		_ = db.Model(&PaymentOrder{}).Where("id = ?", order.Id).Updates(upd).Error
	}
	return order, inv, nil
}

func (p *PaymentService) getOrder(id uint) (*PaymentOrder, error) {
	db := database.GetDB()
	var o PaymentOrder
	if err := db.Where("id = ?", id).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (p *PaymentService) findOrderByPayload(payload string) (*PaymentOrder, error) {
	if payload == "" {
		return nil, gorm.ErrRecordNotFound
	}
	db := database.GetDB()
	var o PaymentOrder
	if err := db.Where("idempotency_key = ?", payload).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (p *PaymentService) markFailed(id uint) {
	db := database.GetDB()
	_ = db.Model(&PaymentOrder{}).Where("id = ? AND status = ?", id, StatusPending).
		Update("status", StatusFailed).Error
}

// ApplyPaidOrder finalizes a pending order and renews the client exactly once.
// The conditional UPDATE ... WHERE status='pending' (checked via RowsAffected)
// is atomic under SQLite write serialization, so concurrent confirmations (a
// redelivered Telegram update or a poll race) are safe no-ops. Returns whether
// a renewal was applied and the bound Telegram user id (for notification).
func (p *PaymentService) ApplyPaidOrder(orderID uint, chargeID string, raw []byte) (bool, int64, error) {
	db := database.GetDB()
	var inboundIds []uint
	var tgUserID int64
	err := db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&PaymentOrder{}).
			Where("id = ? AND status = ?", orderID, StatusPending).
			Updates(map[string]any{
				"status":             StatusPaid,
				"paid_at":            nowUnix(),
				"provider_charge_id": chargeID,
				"provider_payload":   raw,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errAlreadyApplied
		}
		var order PaymentOrder
		if err := tx.Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		var tariff Tariff
		if err := tx.Where("id = ?", order.TariffId).First(&tariff).Error; err != nil {
			return err
		}
		// Zero-value tariffs must never grant a renewal (defense in depth; the
		// purchase path already rejects them).
		if tariff.Price <= 0 && tariff.StarsAmount <= 0 {
			return fmt.Errorf("tariff has no price")
		}
		var client model.Client
		if err := tx.Where("id = ?", order.ClientId).First(&client).Error; err != nil {
			return err
		}
		tgUserID = order.TelegramUserId

		now := nowUnix()
		updates := map[string]any{"enable": true}
		if tariff.AddDays > 0 {
			base := client.Expiry
			if base < now {
				base = now
			}
			updates["expiry"] = base + int64(tariff.AddDays)*86400
		}
		if tariff.AddTrafficBytes > 0 {
			updates["volume"] = client.Volume + tariff.AddTrafficBytes
			updates["total_up"] = client.TotalUp + client.Up
			updates["total_down"] = client.TotalDown + client.Down
			updates["up"] = 0
			updates["down"] = 0
		}
		if err := tx.Model(&model.Client{}).Where("id = ?", client.Id).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.Changes{
			DateTime: now,
			Actor:    "PaidSubBot",
			Key:      "clients",
			Action:   "renew",
			Obj:      jsonString(client.Name),
		}).Error; err != nil {
			return err
		}
		if len(client.Inbounds) > 0 {
			_ = json.Unmarshal(client.Inbounds, &inboundIds)
		}
		return nil
	})
	if errors.Is(err, errAlreadyApplied) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	// Post-commit: re-add the (re-enabled) user to its inbounds in the running
	// core. A restart failure does not roll back the paid renewal (logged).
	if len(inboundIds) > 0 {
		if rErr := (&service.InboundService{}).RestartInbounds(database.GetDB(), inboundIds); rErr != nil {
			logger.Warning("paidsub: restart inbounds after renewal failed: ", rErr)
		}
	}
	_ = (&service.AuditService{}).Record(service.AuditEvent{
		Actor:    "PaidSubBot",
		Event:    "paidsub_paid",
		Resource: "paidsub",
		Severity: service.AuditSeverityInfo,
		Details:  map[string]any{"orderId": orderID},
	})
	return true, tgUserID, nil
}

// ExpireStaleOrders marks pending orders past their TTL as expired.
func (p *PaymentService) ExpireStaleOrders() error {
	db := database.GetDB()
	return db.Model(&PaymentOrder{}).
		Where("status = ? AND expires_at > 0 AND expires_at < ?", StatusPending, nowUnix()).
		Update("status", StatusExpired).Error
}

// ---- order history & refunds ----

// OrdersForTgUser returns the most recent orders belonging to a Telegram user,
// scoped strictly by telegram_user_id (never another user's orders).
func (p *PaymentService) OrdersForTgUser(tgUserId int64, limit int) ([]PaymentOrder, error) {
	if tgUserId <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	db := database.GetDB()
	var orders []PaymentOrder
	if err := db.Where("telegram_user_id = ?", tgUserId).Order("id desc").Limit(limit).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// RefundableOrdersForTgUser returns a user's paid (refundable) orders.
func (p *PaymentService) RefundableOrdersForTgUser(tgUserId int64, limit int) ([]PaymentOrder, error) {
	if tgUserId <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	db := database.GetDB()
	var orders []PaymentOrder
	if err := db.Where("telegram_user_id = ? AND status = ?", tgUserId, StatusPaid).
		Order("id desc").Limit(limit).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// finalizeRefund marks a paid order as refunded exactly once and, when revoke is
// true, rolls back the days/traffic that order granted. The conditional UPDATE
// ... WHERE status='paid' (checked via RowsAffected) makes a double refund a
// safe no-op (returns errAlreadyApplied). Affected inbounds are restarted
// post-commit so the running core re-evaluates the reduced limits. The client is
// never disabled by a refund.
func (p *PaymentService) finalizeRefund(orderID uint, revoke bool) error {
	db := database.GetDB()
	var inboundIds []uint
	err := db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&PaymentOrder{}).
			Where("id = ? AND status = ?", orderID, StatusPaid).
			Update("status", StatusRefunded)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errAlreadyApplied
		}
		if !revoke {
			return nil
		}
		var order PaymentOrder
		if err := tx.Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		var tariff Tariff
		if err := tx.Where("id = ?", order.TariffId).First(&tariff).Error; err != nil {
			return err
		}
		var client model.Client
		if err := tx.Where("id = ?", order.ClientId).First(&client).Error; err != nil {
			return err
		}
		now := nowUnix()
		updates := map[string]any{}
		if tariff.AddDays > 0 && client.Expiry > 0 {
			newExpiry := client.Expiry - int64(tariff.AddDays)*86400
			if newExpiry < now {
				newExpiry = now
			}
			updates["expiry"] = newExpiry
		}
		if tariff.AddTrafficBytes > 0 {
			newVolume := client.Volume - tariff.AddTrafficBytes
			if newVolume < 0 {
				newVolume = 0
			}
			updates["volume"] = newVolume
		}
		if len(updates) == 0 {
			return nil
		}
		if err := tx.Model(&model.Client{}).Where("id = ?", client.Id).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.Changes{
			DateTime: now,
			Actor:    "PaidSubBot",
			Key:      "clients",
			Action:   "refund",
			Obj:      jsonString(client.Name),
		}).Error; err != nil {
			return err
		}
		if len(client.Inbounds) > 0 {
			_ = json.Unmarshal(client.Inbounds, &inboundIds)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(inboundIds) > 0 {
		if rErr := (&service.InboundService{}).RestartInbounds(database.GetDB(), inboundIds); rErr != nil {
			logger.Warning("paidsub: restart inbounds after refund failed: ", rErr)
		}
	}
	_ = (&service.AuditService{}).Record(service.AuditEvent{
		Actor:    "PaidSubBot",
		Event:    "paidsub_refunded",
		Resource: "paidsub",
		Severity: service.AuditSeverityInfo,
		Details:  map[string]any{"orderId": orderID, "revoke": revoke},
	})
	return nil
}

// RefundOrder is the admin-initiated refund (panel Orders tab). For Stars it
// returns the money via refundStarPayment FIRST, then marks the order refunded
// (so the admin can cleanly retry if Telegram rejects the call); for every other
// provider the money must be refunded in the provider's own dashboard, so this
// only marks the order refunded (status "refunded_manual"). revoke is the
// admin's per-refund choice to roll back the granted days/traffic.
func (p *PaymentService) RefundOrder(ctx context.Context, orderID uint, revoke bool) (string, error) {
	order, err := p.getOrder(orderID)
	if err != nil {
		return "", err
	}
	if order.Status != StatusPaid {
		return "", errRefundNotApplicable
	}
	// Defensive: a paid order always has Amount > 0 (CreateOrder rejects zero),
	// so a non-positive amount means a corrupted row — never act on it.
	if order.Amount <= 0 {
		return "", errRefundNotApplicable
	}
	if order.Provider == string(ProviderStars) {
		sender, err := newSenderBot()
		if err != nil {
			return "", err
		}
		charge := strings.TrimPrefix(order.ProviderChargeID, "tg:")
		if charge == "" {
			return "", fmt.Errorf("order has no Stars charge id")
		}
		// An "already refunded" response means a concurrent refund (e.g. the bot
		// path) returned the money first — treat it as success, not a failure.
		if err := sender.refundStarPayment(ctx, order.TelegramUserId, charge); err != nil && !isAlreadyRefunded(err) {
			return "", fmt.Errorf("stars refund failed")
		}
		if err := p.finalizeRefund(orderID, revoke); err != nil && !errors.Is(err, errAlreadyApplied) {
			return "", err
		}
		return "refunded", nil
	}
	if err := p.finalizeRefund(orderID, revoke); err != nil && !errors.Is(err, errAlreadyApplied) {
		return "", err
	}
	return "refunded_manual", nil
}

// ---- bot purchase flow ----

func (b *Bot) cmdBuy(ctx context.Context, chatID int64, tgID int64, l lang) {
	if _, err := b.svc.ClientByTgUserId(tgID); err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	tariffs, _ := b.payments.tariffs.GetEnabled()
	var rows [][]inlineButton
	for i := range tariffs {
		t := tariffs[i]
		if len(b.payments.enabledProvidersForTariff(&t)) == 0 {
			continue
		}
		rows = append(rows, []inlineButton{{Text: tariffButtonLabel(&t), CallbackData: fmt.Sprintf("tariff:%d", t.Id)}})
	}
	if len(rows) == 0 {
		_ = b.sendMessage(ctx, chatID, tr(l, "buy_none"), nil)
		return
	}
	_ = b.sendMessage(ctx, chatID, tr(l, "buy_title"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) handleTariffSelect(ctx context.Context, chatID int64, tgID int64, tariffID uint, l lang) {
	t, err := b.payments.tariffs.Get(tariffID)
	if err != nil || !t.Enabled {
		_ = b.sendMessage(ctx, chatID, tr(l, "buy_none"), nil)
		return
	}
	provs := b.payments.enabledProvidersForTariff(t)
	if len(provs) == 0 {
		_ = b.sendMessage(ctx, chatID, tr(l, "buy_none"), nil)
		return
	}
	if len(provs) == 1 {
		b.startPurchase(ctx, chatID, tgID, t, provs[0], l)
		return
	}
	var rows [][]inlineButton
	for _, prov := range provs {
		rows = append(rows, []inlineButton{{Text: prov.Title(l), CallbackData: fmt.Sprintf("pay:%d:%s", t.Id, prov.Kind())}})
	}
	_ = b.sendMessage(ctx, chatID, tr(l, "buy_choose_provider"), &inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) handlePay(ctx context.Context, chatID int64, tgID int64, tariffID uint, kind string, l lang) {
	t, err := b.payments.tariffs.Get(tariffID)
	if err != nil || !t.Enabled {
		_ = b.sendMessage(ctx, chatID, tr(l, "buy_none"), nil)
		return
	}
	prov := b.payments.providerByKind(ProviderKind(kind))
	if prov == nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "buy_none"), nil)
		return
	}
	b.startPurchase(ctx, chatID, tgID, t, prov, l)
}

func (b *Bot) startPurchase(ctx context.Context, chatID int64, tgID int64, t *Tariff, prov PaymentProvider, l lang) {
	client, err := b.svc.ClientByTgUserId(tgID)
	if err != nil {
		_ = b.sendMessage(ctx, chatID, tr(l, "not_linked"), nil)
		return
	}
	_, inv, err := b.payments.CreateOrder(ctx, client, t, prov.Kind(), tgID)
	if err != nil {
		logger.Warning("paidsub: create order failed: ", err)
		_ = b.sendMessage(ctx, chatID, tr(l, "pay_invoice_failed"), nil)
		return
	}
	switch inv.Method {
	case InvoiceTelegramNative:
		if err := b.sendInvoice(ctx, chatID, inv); err != nil {
			logger.Warning("paidsub: sendInvoice failed: ", err)
			_ = b.sendMessage(ctx, chatID, tr(l, "pay_invoice_failed"), nil)
		}
	case InvoiceURL:
		kb := &inlineKeyboard{InlineKeyboard: [][]inlineButton{{{Text: tr(l, "pay_open"), URL: inv.PayURL}}}}
		_ = b.sendMessage(ctx, chatID, tr(l, "pay_open_hint"), kb)
	case InvoiceManualLink:
		var order *PaymentOrder
		// Re-fetch the freshly created order id for the manual button.
		order, _ = b.payments.findOrderByPayload(inv.Payload)
		var rows [][]inlineButton
		rows = append(rows, []inlineButton{{Text: tr(l, "pay_open"), URL: inv.PayURL}})
		if order != nil {
			rows = append(rows, []inlineButton{{Text: tr(l, "pay_manual_btn"), CallbackData: fmt.Sprintf("paid:%d", order.Id)}})
		}
		_ = b.sendMessage(ctx, chatID, tr(l, "pay_open_hint"), &inlineKeyboard{InlineKeyboard: rows})
	}
}

func (b *Bot) handleManualPaid(ctx context.Context, chatID int64, tgID int64, orderID uint, l lang) {
	order, err := b.payments.getOrder(orderID)
	if err != nil || order.TelegramUserId != tgID {
		return // never act on another user's order
	}
	(&service.TelegramService{}).NotifyTelegramEvent("paidsub_manual_claim", map[string]string{
		"orderId":  fmt.Sprintf("%d", order.Id),
		"clientId": fmt.Sprintf("%d", order.ClientId),
	})
	_ = b.sendMessage(ctx, chatID, tr(l, "pay_manual_sent"), nil)
}

// ---- payment confirmation (Telegram-native) ----

func (b *Bot) handlePreCheckout(ctx context.Context, q *tgPreCheckoutQuery) {
	order, err := b.payments.findOrderByPayload(q.InvoicePayload)
	ok := err == nil &&
		order.Status == StatusPending &&
		q.TotalAmount == order.Amount &&
		strings.EqualFold(q.Currency, order.Currency) &&
		(order.TelegramUserId == 0 || q.From.ID == order.TelegramUserId)
	if ok {
		_ = b.answerPreCheckout(ctx, q.ID, true, "")
		return
	}
	_ = b.answerPreCheckout(ctx, q.ID, false, "Order is no longer valid")
}

func (b *Bot) handleSuccessfulPayment(ctx context.Context, m *tgMessage) {
	if m.From == nil {
		return
	}
	l := pickLang(m.From.LanguageCode)
	sp := m.SuccessfulPayment
	order, err := b.payments.findOrderByPayload(sp.InvoicePayload)
	if err != nil {
		logger.Warning("paidsub: successful_payment for unknown order")
		return
	}
	if sp.TotalAmount != order.Amount || !strings.EqualFold(sp.Currency, order.Currency) {
		logger.Warning("paidsub: payment amount/currency mismatch; refusing renewal")
		b.payments.markFailed(order.Id)
		(&service.TelegramService{}).NotifyTelegramEvent("paidsub_payment_mismatch", map[string]string{
			"orderId": fmt.Sprintf("%d", order.Id),
		})
		return
	}
	// Defence in depth: the payer must be the Telegram user the order was created
	// for (the payload + pending status are the primary gate).
	if order.TelegramUserId != 0 && m.From.ID != order.TelegramUserId {
		logger.Warning("paidsub: successful_payment from unexpected telegram user; refusing renewal")
		b.payments.markFailed(order.Id)
		(&service.TelegramService{}).NotifyTelegramEvent("paidsub_payment_mismatch", map[string]string{
			"orderId": fmt.Sprintf("%d", order.Id),
		})
		return
	}
	charge := sp.TelegramPaymentChargeID
	if charge == "" {
		charge = sp.ProviderPaymentChargeID
	}
	applied, _, err := b.payments.ApplyPaidOrder(order.Id, "tg:"+charge, nil)
	if err != nil {
		logger.Warning("paidsub: apply paid order failed: ", err)
		_ = b.sendMessage(ctx, m.Chat.ID, tr(l, "error"), nil)
		return
	}
	if applied {
		_ = b.sendMessage(ctx, m.Chat.ID, tr(l, "pay_success"), b.menuKeyboard(l))
	}
}

// ---- helpers ----

func tariffButtonLabel(t *Tariff) string {
	price := ""
	switch {
	case t.Price > 0:
		price = fmt.Sprintf("%.2f %s", float64(t.Price)/100.0, t.Currency)
	case t.StarsAmount > 0:
		price = fmt.Sprintf("%d ⭐", t.StarsAmount)
	}
	if price == "" {
		return t.Name
	}
	return fmt.Sprintf("%s — %s", t.Name, price)
}

// formatOrderAmount renders an order amount: Telegram Stars (XTR) are whole
// units; every other currency is stored in minor units (e.g. kopeks/cents).
func formatOrderAmount(amount int64, currency string) string {
	if currency == "XTR" {
		return fmt.Sprintf("%d ⭐", amount)
	}
	return fmt.Sprintf("%.2f %s", float64(amount)/100.0, currency)
}

// jsonString marshals s as a JSON string so a client name containing quotes or
// backslashes cannot corrupt the Changes.Obj payload.
func jsonString(s string) json.RawMessage {
	b, err := json.Marshal(s)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return b
}
