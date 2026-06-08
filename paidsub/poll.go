package paidsub

import (
	"context"
	"sync"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
)

var pollMu sync.Mutex

// PollOnce expires stale pending orders and polls out-of-band providers
// (CryptoBot) for confirmations. It is single-flight: overlapping ticks are
// skipped so a paid invoice is never applied twice (the RowsAffected guard +
// partial unique index are the final defense).
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
	if err := ps.ExpireStaleOrders(); err != nil {
		logger.Warning("paidsub: expire stale orders: ", err)
	}

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
