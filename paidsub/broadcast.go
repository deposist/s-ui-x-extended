package paidsub

import (
	"context"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/util/common"
)

// broadcastThrottle paces sends to stay well under Telegram's ~30 msg/s limit.
const broadcastThrottle = 60 * time.Millisecond

// Broadcast sends a custom announcement to every bound Telegram user. It runs
// sequentially with a small throttle. Returns counts of sent/failed messages.
func Broadcast(ctx context.Context, text string) (sent int, failed int, err error) {
	text = truncateRunes(text, 4096)
	if text == "" {
		return 0, 0, common.NewError("message is empty")
	}
	b, err := newSenderBot()
	if err != nil {
		return 0, 0, err
	}
	db := database.GetDB()
	var rows []Binding
	if err := db.Find(&rows).Error; err != nil {
		return 0, 0, err
	}
	for _, r := range rows {
		if r.TgUserId <= 0 {
			continue
		}
		if sendErr := b.sendMessage(ctx, r.TgUserId, text, nil); sendErr != nil {
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
