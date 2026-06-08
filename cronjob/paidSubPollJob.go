package cronjob

import (
	"context"

	"github.com/deposist/s-ui-x/paidsub"
)

// PaidSubPollJob drives the experimental Paid Subscriptions out-of-band payment
// poll (CryptoBot) and stale-order expiry. It self-gates on paidSubEnabled.
type PaidSubPollJob struct{}

func NewPaidSubPollJob() *PaidSubPollJob {
	return &PaidSubPollJob{}
}

func (j *PaidSubPollJob) Run() {
	paidsub.PollOnce(context.Background())
}
