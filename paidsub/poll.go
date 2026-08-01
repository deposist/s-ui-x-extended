package paidsub

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"

	"gorm.io/gorm"
)

var pollMu sync.Mutex

const (
	cryptoBotPollGraceSeconds = int64(24 * 60 * 60)
	cryptoBotReconcileBatch   = 100
	cryptoBotPollMaxBatches   = 10
)

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
	// 2. Confirm out-of-band payments in bounded provider requests. CryptoBot
	// orders are expired only when that request authoritatively reports a
	// terminal invoice; an outage/cancel never changes potentially-paid rows.
	pollCryptoBot(ctx, ps)

	// 3. Expire non-polled providers on the short order TTL.
	if ctx.Err() == nil {
		if err := ps.ExpireStaleOrders(); err != nil {
			logger.Warning("paidsub: expire stale orders: ", err)
		}
	}
}

func reconcileCryptoBot(ctx context.Context, ps *PaymentService) {
	prov := ps.providerByKind(ProviderCryptoBot)
	reconciler, ok := prov.(invoiceReconciler)
	if !ok {
		return
	}
	retryCryptoBotSiblingCancellations(ctx, prov)
	if ctx.Err() != nil {
		return
	}
	var unresolved []PaymentOrder
	db := database.GetDB()
	query := db.Where("provider = ? AND status IN ?", string(ProviderCryptoBot), []string{StatusInvoiceCreating, StatusRecoverable}).
		Order("id ASC").Limit(cryptoBotReconcileBatch)
	if cursor := loadPollCursor(db, "cryptobot:reconcile"); cursor > 0 {
		query = query.Where("id > ?", cursor)
	}
	if err := query.Find(&unresolved).Error; err != nil {
		logger.Warning("paidsub: reconcile load unresolved: ", err)
		return
	}
	if len(unresolved) == 0 {
		if loadPollCursor(db, "cryptobot:reconcile") == 0 {
			return
		}
		storePollCursor(db, "cryptobot:reconcile", 0)
		if err := db.Where("provider = ? AND status IN ?", string(ProviderCryptoBot), []string{StatusInvoiceCreating, StatusRecoverable}).
			Order("id ASC").Limit(cryptoBotReconcileBatch).Find(&unresolved).Error; err != nil {
			logger.Warning("paidsub: reconcile load wrapped unresolved: ", err)
			return
		}
	}
	if len(unresolved) == 0 {
		return
	}
	results, err := reconciler.ReconcileInvoices(ctx, unresolved)
	if err != nil {
		logger.Warning("paidsub: cryptobot reconcile: ", err)
		return
	}
	resultsByOrder := make(map[uint][]ReconciledInvoice)
	orderIDs := make([]uint, 0)
	for _, result := range results {
		if _, ok := resultsByOrder[result.OrderID]; !ok {
			orderIDs = append(orderIDs, result.OrderID)
		}
		resultsByOrder[result.OrderID] = append(resultsByOrder[result.OrderID], result)
	}
	var lastHandled uint
	for _, orderID := range orderIDs {
		orderResults := resultsByOrder[orderID]
		var order PaymentOrder
		if err := database.GetDB().Where("id = ? AND status IN ?", orderID, []string{StatusInvoiceCreating, StatusRecoverable}).First(&order).Error; err != nil {
			if database.IsNotFound(err) {
				lastHandled = orderID
				continue
			}
			logger.Warning("paidsub: load reconciled invoice order: ", err)
			break
		}
		if order.SnapshotVersion != paymentOrderSnapshotVersion {
			reconcileLegacyCryptoBotOrder(ctx, prov, order, orderResults)
			lastHandled = orderID
			continue
		}
		selected, ok := selectReconciledInvoice(orderResults)
		if !ok {
			if err := database.GetDB().Model(&PaymentOrder{}).Where("id = ?", orderID).Update("status", StatusRecoverable).Error; err != nil {
				logger.Warning("paidsub: persist missing reconciled invoice: ", err)
				break
			}
			lastHandled = orderID
			continue
		}
		if selected.Paid {
			if err := applyReconciledPaidCryptoBot(ctx, ps, prov, order, selected, orderResults); err != nil {
				logger.Warning("paidsub: apply reconciled invoice: ", err)
				break
			}
			lastHandled = orderID
			continue
		}
		res := database.GetDB().Model(&PaymentOrder{}).
			Where("id = ? AND status IN ?", selected.OrderID, []string{StatusInvoiceCreating, StatusRecoverable}).
			Updates(map[string]any{"provider_ref": selected.ProviderRef, "external_url": selected.PayURL, "status": StatusPending})
		if res.Error != nil {
			logger.Warning("paidsub: save reconciled invoice: ", res.Error)
			break
		}
		lastHandled = orderID
	}
	if lastHandled > 0 {
		storePollCursor(db, "cryptobot:reconcile", lastHandled)
	}
}

func applyReconciledPaidCryptoBot(ctx context.Context, ps *PaymentService, prov PaymentProvider, order PaymentOrder, paid ReconciledInvoice, results []ReconciledInvoice) error {
	refs := make([]string, 0)
	for _, result := range results {
		if result.ProviderStatus == "active" && result.ProviderRef != "" && result.ProviderRef != paid.ProviderRef {
			refs = append(refs, result.ProviderRef)
		}
	}
	sort.Strings(refs)
	payload, _ := json.Marshal(map[string]string{"ref": paid.ProviderRef})
	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&PaymentOrder{}).
			Where("id = ? AND status IN ?", order.Id, []string{StatusInvoiceCreating, StatusRecoverable}).
			Updates(map[string]any{
				"provider_ref": paid.ProviderRef, "provider_payload": payload,
				"external_url": paid.PayURL, "status": StatusPending,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errAlreadyApplied
		}
		for _, ref := range refs {
			if err := tx.Exec(`INSERT OR IGNORE INTO paidsub_invoice_cancellations(order_id, provider, provider_ref)
				VALUES (?, ?, ?)`, order.Id, string(ProviderCryptoBot), ref).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if _, _, err := ps.ApplyPaidOrder(order.Id, paid.ProviderChargeID, nil); err != nil {
		return err
	}
	deleter, ok := prov.(invoiceDeleter)
	if !ok {
		return nil
	}
	for _, ref := range refs {
		if err := deleter.DeleteInvoice(ctx, ref); err != nil {
			return err
		}
		if err := forgetSiblingCancellation(order.Id, ref); err != nil {
			return err
		}
	}
	return nil
}

func forgetSiblingCancellation(orderID uint, ref string) error {
	return database.GetDB().Exec(`DELETE FROM paidsub_invoice_cancellations WHERE order_id = ? AND provider = ? AND provider_ref = ?`,
		orderID, string(ProviderCryptoBot), ref).Error
}

func retryCryptoBotSiblingCancellations(ctx context.Context, prov PaymentProvider) {
	deleter, ok := prov.(invoiceDeleter)
	if !ok {
		return
	}
	var pending []struct {
		OrderID     uint   `gorm:"column:order_id"`
		ProviderRef string `gorm:"column:provider_ref"`
	}
	if err := database.GetDB().Raw(`SELECT order_id, provider_ref FROM paidsub_invoice_cancellations
		WHERE provider = ? ORDER BY order_id, provider_ref LIMIT 100`, string(ProviderCryptoBot)).Scan(&pending).Error; err != nil {
		logger.Warning("paidsub: load sibling invoice cancellations: ", err)
		return
	}
	for _, item := range pending {
		if err := deleter.DeleteInvoice(ctx, item.ProviderRef); err != nil {
			logger.Warning("paidsub: retry sibling invoice cancellation: ", err)
			continue
		}
		if err := forgetSiblingCancellation(item.OrderID, item.ProviderRef); err != nil {
			logger.Warning("paidsub: clear sibling invoice cancellation: ", err)
			continue
		}
	}
}
func loadPollCursor(db *gorm.DB, key string) uint {
	var cursor uint
	_ = db.Raw(`SELECT last_order_id FROM paidsub_poll_cursors WHERE provider = ?`, key).Scan(&cursor).Error
	return cursor
}

func storePollCursor(db *gorm.DB, key string, cursor uint) {
	if err := db.Exec(`INSERT INTO paidsub_poll_cursors(provider, last_order_id) VALUES (?, ?)
		ON CONFLICT(provider) DO UPDATE SET last_order_id = excluded.last_order_id`, key, cursor).Error; err != nil {
		logger.Warning("paidsub: store poll cursor: ", err)
	}
}

func selectReconciledInvoice(results []ReconciledInvoice) (ReconciledInvoice, bool) {
	for _, result := range results {
		if result.Paid && !result.MetadataMismatch {
			return result, true
		}
	}
	for _, result := range results {
		if result.ProviderStatus == "active" && !result.MetadataMismatch {
			return result, true
		}
	}
	return ReconciledInvoice{}, false
}

func reconcileLegacyCryptoBotOrder(ctx context.Context, prov PaymentProvider, order PaymentOrder, results []ReconciledInvoice) {
	refs := make(map[string]struct{}, len(results)+1)
	paid := false
	for _, result := range results {
		if result.Paid || (result.MetadataMismatch && result.ProviderStatus == "paid") {
			paid = true
			logger.Warning("paidsub: paid legacy CryptoBot invoice requires manual review; order ", order.Id)
		}
		status := result.ProviderStatus
		if status == "" && !result.Paid {
			status = "active"
		}
		if status == "active" && result.ProviderRef != "" {
			refs[result.ProviderRef] = struct{}{}
		}
	}
	if !paid && order.ProviderRef != "" {
		storedRefTerminal := false
		for _, result := range results {
			if result.ProviderRef == order.ProviderRef && (result.ProviderStatus == "deleted" || result.ProviderStatus == "expired") {
				storedRefTerminal = true
				break
			}
		}
		if !storedRefTerminal {
			refs[order.ProviderRef] = struct{}{}
		}
	}
	if paid && len(refs) == 0 {
		return
	}
	if len(refs) == 0 {
		terminal := len(results) > 0
		for _, result := range results {
			if result.ProviderStatus != "missing" && result.ProviderStatus != "deleted" && result.ProviderStatus != "expired" {
				terminal = false
				break
			}
		}
		if terminal {
			if err := database.GetDB().Model(&PaymentOrder{}).
				Where("id = ? AND status = ?", order.Id, StatusRecoverable).
				Updates(map[string]any{
					"status": StatusFailed, "external_url": "", "provider_ref": "", "provider_payload": nil,
					"snapshot_version": paymentOrderLegacyResolvedVersion,
				}).Error; err != nil {
				logger.Warning("paidsub: mark terminal legacy CryptoBot invoice resolved: ", err)
			}
		}
		return
	}
	deleter, ok := prov.(invoiceDeleter)
	if !ok {
		logger.Warning("paidsub: legacy CryptoBot invoice cannot be cancelled; order ", order.Id)
		return
	}
	orderedRefs := make([]string, 0, len(refs))
	for ref := range refs {
		orderedRefs = append(orderedRefs, ref)
	}
	sort.Strings(orderedRefs)
	for i, ref := range orderedRefs {
		if err := deleter.DeleteInvoice(ctx, ref); err != nil {
			logger.Warning("paidsub: cancel legacy CryptoBot invoice: ", err)
			return
		}
		if i+1 < len(orderedRefs) {
			remaining := orderedRefs[i+1]
			payload, _ := json.Marshal(map[string]string{"ref": remaining})
			if err := database.GetDB().Model(&PaymentOrder{}).
				Where("id = ? AND status = ?", order.Id, StatusRecoverable).
				Updates(map[string]any{"provider_ref": remaining, "provider_payload": payload, "external_url": ""}).Error; err != nil {
				logger.Warning("paidsub: save remaining legacy CryptoBot invoice: ", err)
				return
			}
		}
	}
	if paid {
		return
	}
	if err := database.GetDB().Model(&PaymentOrder{}).
		Where("id = ? AND status = ?", order.Id, StatusRecoverable).
		Updates(map[string]any{
			"status": StatusFailed, "external_url": "", "provider_ref": "", "provider_payload": nil,
			"snapshot_version": paymentOrderLegacyResolvedVersion,
		}).Error; err != nil {
		logger.Warning("paidsub: mark legacy CryptoBot invoice cancelled: ", err)
	}
}

// pollCryptoBot visits pending orders by increasing id and wraps within the same
// tick. Every provider request is bounded; successful batches are committed
// independently, while a failed/canceled batch leaves all of its orders pending.
func pollCryptoBot(ctx context.Context, ps *PaymentService) {
	prov := ps.providerByKind(ProviderCryptoBot)
	poller, ok := prov.(pollingProvider)
	if !ok {
		return
	}

	db := database.GetDB()
	cursorKey := "cryptobot:poll"
	cursor := loadPollCursor(db, cursorKey)
	for range cryptoBotPollMaxBatches {
		if ctx.Err() != nil {
			return
		}
		var pending []PaymentOrder
		if err := db.Where("provider = ? AND status = ? AND id > ?", string(ProviderCryptoBot), StatusPending, cursor).
			Order("id ASC").Limit(cryptoBotPollBatchSize).Find(&pending).Error; err != nil {
			logger.Warning("paidsub: poll load pending: ", err)
			return
		}
		if len(pending) == 0 {
			storePollCursor(db, cursorKey, 0)
			return
		}
		batchCursor := pending[len(pending)-1].Id
		outcome, err := poller.Poll(ctx, pending)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Warning("paidsub: cryptobot poll: ", err)
			cursor = batchCursor
			continue
		}
		batchHandled := true
		for _, result := range outcome.Paid {
			applied, tgID, err := ps.ApplyPaidOrder(result.OrderID, result.ProviderChargeID, result.RawPayload)
			if err != nil {
				logger.Warning("paidsub: apply polled order: ", err)
				batchHandled = false
				break
			}
			if applied && tgID > 0 {
				notifyPaid(ctx, tgID)
			}
		}
		if !batchHandled || ctx.Err() != nil {
			return
		}
		if len(outcome.MissingOrderIDs) > 0 {
			res := db.Model(&PaymentOrder{}).
				Where("id IN ? AND status = ?", outcome.MissingOrderIDs, StatusPending).
				Update("status", StatusRecoverable)
			if res.Error != nil {
				logger.Warning("paidsub: persist missing CryptoBot invoices: ", res.Error)
				return
			}
		}
		if err := ps.ExpireTerminalPolledOrders(outcome.TerminalOrderIDs, cryptoBotPollGraceSeconds); err != nil {
			logger.Warning("paidsub: expire terminal CryptoBot orders: ", err)
			return
		}
		cursor = batchCursor
		storePollCursor(db, cursorKey, cursor)
		if len(pending) < cryptoBotPollBatchSize {
			storePollCursor(db, cursorKey, 0)
			return
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
