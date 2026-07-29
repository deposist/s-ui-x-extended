package paidsub

import (
	"context"
	"sync"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
)

var pollMu sync.Mutex

// cryptoBotPollGraceSeconds is the long hard-TTL after which an abandoned
// (never-paid) CryptoBot order is reaped. Polled orders are deliberately
// excluded from the short order-TTL ExpireStaleOrders (see PollOnce) so a
// payment confirmed out-of-band AFTER the local TTL is still caught by the next
// poll; this generous window only cleans up invoices that were never paid.
const cryptoBotPollGraceSeconds int64 = 24 * 60 * 60

// PollOnce polls out-of-band providers (CryptoBot) for confirmations and then
// expires stale pending orders. It is single-flight: overlapping ticks are
// skipped so a paid invoice is never applied twice (the RowsAffected guard +
// partial unique index are the final defense).
//
// Ordering matters: the poll runs BEFORE any expiry pass so a payment confirmed
// after the local order TTL is applied before it could be moved out of the
// pending set - otherwise a late-but-valid payment would be silently lost
// (money taken, no grant, no recovery).
func PollOnce(ctx context.Context) {
	setting := service.SettingService{}
	if enabled, err := setting.GetPaidSubEnabled(); err != nil || !enabled {
		return
	}
	if !pollMu.TryLock() {
		return
	}
	defer pollMu.Unlock()

	ps := NewPaymentService()

	// 1. Recover provider invoices created before their local reference was saved.
	reconcileCryptoBot(ctx, ps)
	// 2. Confirm out-of-band payments first (before any expiry can hide them).
	pollCryptoBot(ctx, ps)

	// 3. Expire non-polled providers on the short order TTL.
	if err := ps.ExpireStaleOrders(); err != nil {
		logger.Warning("paidsub: expire stale orders: ", err)
	}
	// 4. Reap abandoned CryptoBot invoices only after a long grace window.
	if err := ps.ExpireStalePolledOrders(cryptoBotPollGraceSeconds); err != nil {
		logger.Warning("paidsub: expire stale polled orders: ", err)
	}
}

func reconcileCryptoBot(ctx context.Context, ps *PaymentService) {
	prov := ps.providerByKind(ProviderCryptoBot)
	reconciler, ok := prov.(invoiceReconciler)
	if !ok {
		return
	}
	var unresolved []PaymentOrder
	if err := database.GetDB().Where("provider = ? AND status IN ?", string(ProviderCryptoBot), []string{StatusInvoiceCreating, StatusRecoverable}).
		Order("created_at ASC, id ASC").Limit(100).Find(&unresolved).Error; err != nil {
		logger.Warning("paidsub: reconcile load unresolved: ", err)
		return
	}
	results, err := reconciler.ReconcileInvoices(ctx, unresolved)
	if err != nil {
		logger.Warning("paidsub: cryptobot reconcile: ", err)
		return
	}
	for _, result := range results {
		res := database.GetDB().Model(&PaymentOrder{}).
			Where("id = ? AND status IN ?", result.OrderID, []string{StatusInvoiceCreating, StatusRecoverable}).
			Updates(map[string]any{"provider_ref": result.ProviderRef, "external_url": result.PayURL, "status": StatusPending})
		if res.Error != nil || res.RowsAffected != 1 {
			if res.Error != nil {
				logger.Warning("paidsub: save reconciled invoice: ", res.Error)
			}
			continue
		}
		if result.Paid {
			if _, _, err := ps.ApplyPaidOrder(result.OrderID, result.ProviderChargeID, nil); err != nil {
				logger.Warning("paidsub: apply reconciled invoice: ", err)
			}
		}
	}
}

// pollCryptoBot loads pending CryptoBot orders and applies any the provider
// reports as paid. Errors are logged and swallowed (best-effort per tick).
func pollCryptoBot(ctx context.Context, ps *PaymentService) {
	prov := ps.providerByKind(ProviderCryptoBot)
	if prov == nil {
		return
	}
	poller, ok := prov.(pollingProvider)
	if !ok {
		return
	}

	db := database.GetDB()
	var pending []PaymentOrder
	if err := db.Where("provider = ? AND status = ?", string(ProviderCryptoBot), StatusPending).
		Find(&pending).Error; err != nil {
		logger.Warning("paidsub: poll load pending: ", err)
		return
	}
	if len(pending) == 0 {
		return
	}
	results, err := poller.Poll(ctx, pending)
	if err != nil {
		logger.Warning("paidsub: cryptobot poll: ", err)
		return
	}
	for _, r := range results {
		applied, tgID, err := ps.ApplyPaidOrder(r.OrderID, r.ProviderChargeID, r.RawPayload)
		if err != nil {
			logger.Warning("paidsub: apply polled order: ", err)
			continue
		}
		if applied && tgID > 0 {
			notifyPaid(ctx, tgID)
		}
	}
}

func notifyPaid(ctx context.Context, tgUserID int64) {
	b, err := newSenderBot()
	if err != nil {
		return
	}
	_ = b.sendMessage(ctx, tgUserID, tr(langEN, "pay_success"), nil)
}
