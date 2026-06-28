package cronjob

import (
	"context"

	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
)

// CertRenewJob re-issues the managed IP certificate when it nears expiry. It is
// a cheap no-op when auto-renew is disabled or the certificate is still fresh,
// so it is safe to run on a fixed schedule.
type CertRenewJob struct {
	service.IpCertificateService
	ctx context.Context
}

func NewCertRenewJob(ctx context.Context) *CertRenewJob {
	return &CertRenewJob{
		IpCertificateService: service.IpCertificateService{
			Runtime:  service.DefaultRuntime(),
			Settings: &service.SettingService{},
		},
		ctx: ctx,
	}
}

func (j *CertRenewJob) Run() {
	ctx := j.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	renewed, err := j.IpCertificateService.RenewIfNeeded(ctx)
	if err != nil {
		logger.Warning("ip cert renew failed: ", err)
		return
	}
	if renewed {
		logger.Info("ip cert renewed")
	}
}
