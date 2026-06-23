package service

import (
	"context"
	"time"

	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// IpCertificateService issues and renews a Let's Encrypt TLS certificate for a
// bare IP address (RFC 8738 / shortlived profile) entirely in-process via
// go-acme/lego, then applies it to the panel HTTPS listener or an inbound TLS
// profile. All network/ACME code sits behind the acmeIssuer seam so the
// orchestration is unit-testable without touching Let's Encrypt.
type IpCertificateService struct {
	Runtime  *Runtime
	Settings *SettingService
	acme     acmeIssuer
	now      func() time.Time
}

// IpCertStatus is the read model returned to the API/frontend.
type IpCertStatus struct {
	Enabled       bool    `json:"enabled"`
	TargetIP      string  `json:"targetIp"`
	ApplyTarget   string  `json:"applyTarget"`
	Issued        bool    `json:"issued"`
	NotAfter      string  `json:"notAfter"`
	LastIssue     string  `json:"lastIssue"`
	DaysRemaining float64 `json:"daysRemaining"`
	CertPath      string  `json:"certPath"`
}

func (s *IpCertificateService) settings() *SettingService {
	if s.Settings != nil {
		return s.Settings
	}
	return &SettingService{}
}

func (s *IpCertificateService) issuer() acmeIssuer {
	if s.acme != nil {
		return s.acme
	}
	return legoIssuer{}
}

func (s *IpCertificateService) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// IssueNow obtains a fresh certificate for ip and applies it to applyTarget.
// hostname is the request host used for inbound link regeneration ("" is fine
// for the panel target and for cron-driven renewals). Used by the renewal cron
// running inside the live panel, so a "panel" target also triggers a panel
// restart to reload the web certificate.
func (s *IpCertificateService) IssueNow(ctx context.Context, ip, email string, port int, applyTarget, hostname string) (IpCertStatus, error) {
	certPath, keyPath, err := s.obtainAndPersist(ctx, ip, email, port, applyTarget)
	if err != nil {
		return IpCertStatus{}, err
	}

	if err := s.applyToTarget(applyTarget, certPath, keyPath, hostname); err != nil {
		// The certificate is issued and persisted; surface the apply failure
		// but keep the stored state so a later retry/renewal can re-apply.
		status, _ := s.GetStatus()
		return status, common.NewError("ip cert: issued but apply failed: ", err.Error())
	}

	return s.GetStatus()
}

// IssueForCLI obtains a certificate from a one-shot CLI process and points the
// panel HTTPS cert settings at the new files. Unlike IssueNow it never restarts
// the panel - a CLI invocation has no live runtime, and the management script
// (s-ui.sh) stops the panel before issuance and starts it afterwards, which is
// what reloads the web certificate. The apply target is always the panel.
func (s *IpCertificateService) IssueForCLI(ctx context.Context, ip, email string, port int) (IpCertStatus, error) {
	certPath, keyPath, err := s.obtainAndPersist(ctx, ip, email, port, "panel")
	if err != nil {
		return IpCertStatus{}, err
	}

	if err := s.setPanelCertSettings(certPath, keyPath); err != nil {
		// Certificate is issued and persisted; report the apply failure but keep
		// the stored state so the cron/CLI can re-apply later.
		status, _ := s.GetStatus()
		return status, common.NewError("ip cert: issued but apply failed: ", err.Error())
	}

	return s.GetStatus()
}

// obtainAndPersist validates the request, obtains the certificate through the
// ACME issuer, writes the cert/key files, and persists the issued state plus the
// (possibly newly created) ACME account. It returns the on-disk cert/key paths
// so the caller can apply them. It performs no apply itself.
func (s *IpCertificateService) obtainAndPersist(ctx context.Context, ip, email string, port int, applyTarget string) (certPath, keyPath string, err error) {
	if err := validateIssuableIP(ip); err != nil {
		return "", "", err
	}
	if err := validateIpCertApplyTarget(applyTarget); err != nil {
		return "", "", err
	}
	if err := validateIpCertEmail(email, false); err != nil {
		return "", "", err
	}
	if port <= 0 {
		port = 80
	}
	if err := validateIpCertPort(port); err != nil {
		return "", "", err
	}

	set := s.settings()
	accountKey, err := set.getIpCertAccountKey()
	if err != nil {
		return "", "", err
	}
	registrationJSON, err := set.getIpCertAccountRegistration()
	if err != nil {
		return "", "", err
	}

	result, err := s.issuer().Obtain(ctx, acmeRequest{
		IP:               ip,
		Email:            email,
		ChallengePort:    port,
		AccountKeyPEM:    accountKey,
		RegistrationJSON: registrationJSON,
	})
	if err != nil {
		return "", "", common.NewError("ip cert: issuance failed: ", err.Error())
	}

	certPath, keyPath, err = writeCertFiles(ip, result.CertPEM, result.KeyPEM)
	if err != nil {
		return "", "", err
	}
	notAfter, err := parseCertNotAfter(result.CertPEM)
	if err != nil {
		return "", "", err
	}

	if err := s.persistIssued(ipCertIssued{
		ip: ip, email: email, port: port, applyTarget: applyTarget,
		result: result, certPath: certPath, keyPath: keyPath, notAfter: notAfter,
	}); err != nil {
		return "", "", err
	}

	return certPath, keyPath, nil
}

// ipCertIssued bundles everything persisted after a successful issuance.
type ipCertIssued struct {
	ip          string
	email       string
	port        int
	applyTarget string
	result      acmeResult
	certPath    string
	keyPath     string
	notAfter    time.Time
}

// persistIssued stores the issued certificate state, the request parameters
// (so a direct-API issue stays renewable), and the (possibly newly created)
// ACME account so renewals reuse it.
func (s *IpCertificateService) persistIssued(d ipCertIssued) error {
	set := s.settings()
	if d.result.AccountKeyPEM != "" {
		if err := set.setIpCertAccountKey(d.result.AccountKeyPEM); err != nil {
			return err
		}
	}
	if d.result.RegistrationJSON != "" {
		if err := set.setIpCertAccountRegistration(d.result.RegistrationJSON); err != nil {
			return err
		}
	}
	writes := []func() error{
		func() error { return set.setIpCertTargetIP(d.ip) },
		func() error { return set.setIpCertEmail(d.email) },
		func() error { return set.setIpCertChallengePort(d.port) },
		func() error { return set.setIpCertApplyTarget(d.applyTarget) },
		func() error { return set.setIpCertLastIP(d.ip) },
		func() error { return set.setIpCertCertPath(d.certPath) },
		func() error { return set.setIpCertKeyPath(d.keyPath) },
		func() error { return set.setIpCertNotAfter(d.notAfter.UTC().Format(time.RFC3339)) },
		func() error { return set.setIpCertLastIssue(s.clock().UTC().Format(time.RFC3339)) },
	}
	for _, w := range writes {
		if err := w(); err != nil {
			return err
		}
	}
	return nil
}

// RenewIfNeeded re-issues the managed certificate when auto-renew is enabled
// and the remaining validity has dropped below the threshold. It reuses the
// stored target/email/port/apply-target and the persisted ACME account.
func (s *IpCertificateService) RenewIfNeeded(ctx context.Context) (bool, error) {
	set := s.settings()
	enabled, err := set.GetIpCertEnabled()
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}
	ip, err := set.GetIpCertTargetIP()
	if err != nil {
		return false, err
	}
	if ip == "" {
		return false, nil
	}

	// Force a re-issue when the configured target IP no longer matches the IP
	// the stored certificate was issued for (the operator changed ipCertTargetIP
	// in settings): the on-disk cert's SAN would otherwise be wrong until expiry.
	// A blank lastIP means "no prior issue on record" and falls back to the
	// expiry-only decision.
	lastIP, err := set.getIpCertLastIP()
	if err != nil {
		return false, err
	}
	ipChanged := lastIP != "" && lastIP != ip

	notAfter := s.storedNotAfter()
	if !ipChanged && !shouldRenew(notAfter, s.clock()) {
		return false, nil
	}

	email, err := set.GetIpCertEmail()
	if err != nil {
		return false, err
	}
	port, err := set.GetIpCertChallengePort()
	if err != nil {
		return false, err
	}
	applyTarget, err := set.GetIpCertApplyTarget()
	if err != nil {
		return false, err
	}

	if _, err := s.IssueNow(ctx, ip, email, port, applyTarget, ""); err != nil {
		return false, err
	}
	return true, nil
}

// GetStatus reports the current managed-certificate state for the UI.
func (s *IpCertificateService) GetStatus() (IpCertStatus, error) {
	set := s.settings()
	status := IpCertStatus{}
	var err error
	if status.Enabled, err = set.GetIpCertEnabled(); err != nil {
		return IpCertStatus{}, err
	}
	if status.TargetIP, err = set.GetIpCertTargetIP(); err != nil {
		return IpCertStatus{}, err
	}
	if status.ApplyTarget, err = set.GetIpCertApplyTarget(); err != nil {
		return IpCertStatus{}, err
	}
	if status.CertPath, err = set.GetIpCertCertPath(); err != nil {
		return IpCertStatus{}, err
	}
	if status.LastIssue, err = set.GetIpCertLastIssue(); err != nil {
		return IpCertStatus{}, err
	}
	notAfter := s.storedNotAfter()
	if !notAfter.IsZero() {
		status.Issued = true
		status.NotAfter = notAfter.UTC().Format(time.RFC3339)
		status.DaysRemaining = notAfter.Sub(s.clock()).Hours() / 24
	}
	return status, nil
}

// storedNotAfter parses the persisted expiry; an unparseable/empty value yields
// the zero time (treated as "renew/unknown").
func (s *IpCertificateService) storedNotAfter() time.Time {
	raw, err := s.settings().GetIpCertNotAfter()
	if err != nil || raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		logger.Warning("ip cert: stored notAfter is unparseable: ", raw)
		return time.Time{}
	}
	return parsed
}
