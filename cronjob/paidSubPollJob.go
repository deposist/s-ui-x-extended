package cronjob

import (
	"context"

	"github.com/deposist/s-ui-x-extended/paidsub"
)

// PaidSubPollJob drives the experimental Paid Subscriptions out-of-band payment
// poll (CryptoBot) and stale-order expiry. It self-gates on paidSubEnabled.
type PaidSubPollJob struct {
	ctx context.Context
}

func NewPaidSubPollJob(ctx context.Context) *PaidSubPollJob {
	return &PaidSubPollJob{ctx: ctx}
}

func (j *PaidSubPollJob) Run() {
	ctx := j.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	paidsub.PollOnce(ctx)
}
