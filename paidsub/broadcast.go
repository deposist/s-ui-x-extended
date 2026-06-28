package paidsub

import (
	"context"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/util/common"
	"gorm.io/gorm"
)

// broadcastThrottle paces sends to stay well under Telegram's ~30 msg/s limit.
const broadcastThrottle = 60 * time.Millisecond

type broadcastRecipientCandidate struct {
	TgUserId      int64
	ClientEnabled bool
	ClientExpiry  int64
}

func isActiveBroadcastRecipient(candidate broadcastRecipientCandidate, now int64) bool {
	return candidate.TgUserId > 0 && candidate.ClientEnabled && (candidate.ClientExpiry == 0 || candidate.ClientExpiry > now)
}

func activeBroadcastRecipientIDs(db *gorm.DB, now int64) ([]int64, error) {
	var rows []broadcastRecipientCandidate
	if err := db.Table("paidsub_bindings b").
		Select("b.tg_user_id AS tg_user_id, COALESCE(c.enable, 0) AS client_enabled, COALESCE(c.expiry, 0) AS client_expiry").
		Joins("JOIN clients c ON c.id = b.client_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	recipients := make([]int64, 0, len(rows))
	for _, row := range rows {
		if isActiveBroadcastRecipient(row, now) {
			recipients = append(recipients, row.TgUserId)
		}
	}
	return recipients, nil
}

// Broadcast sends a custom announcement to every active bound Telegram user. It
// runs sequentially with a small throttle. Returns counts of sent/failed messages.
func Broadcast(ctx context.Context, text string) (sent int, failed int, err error) {
	text = truncateRunes(text, 4096)
	if text == "" {
		return 0, 0, common.NewError("message is empty")
	}
	b, err := newSenderBot()
	if err != nil {
		return 0, 0, err
	}
	recipients, err := activeBroadcastRecipientIDs(database.GetDB(), nowUnix())
	if err != nil {
		return 0, 0, err
	}
	for _, tgUserId := range recipients {
		if sendErr := b.sendMessage(ctx, tgUserId, text, nil); sendErr != nil {
			failed++
		} else {
			sent++
		}
		select {
		case <-ctx.Done():
			return sent, failed, ctx.Err()
		case <-time.After(broadcastThrottle):
		}
	}
	return sent, failed, nil
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
