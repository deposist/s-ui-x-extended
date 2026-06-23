package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeIssuer is an acmeIssuer stub that records the request it received and
// returns canned PEM, so IssueNow can be exercised without Let's Encrypt.
type fakeIssuer struct {
	calls         int
	lastReq       acmeRequest
	certPEM       []byte
	keyPEM        []byte
	accountKeyOut string
}

func (f *fakeIssuer) Obtain(_ context.Context, req acmeRequest) (acmeResult, error) {
	f.calls++
	f.lastReq = req
	accountKey := req.AccountKeyPEM
	if accountKey == "" {
		accountKey = f.accountKeyOut
	}
	return acmeResult{
		CertPEM:          f.certPEM,
		KeyPEM:           f.keyPEM,
		AccountKeyPEM:    accountKey,
		RegistrationJSON: `{"uri":"https://acme.test/acct/1"}`,
	}, nil
}

func newIpCertTestService(t *testing.T, issuer acmeIssuer, now time.Time) *IpCertificateService {
	t.Helper()
	initSettingTestDB(t)
	rt := NewRuntimeWithCoreProvider(nil)
	// Cancel any restart the panel-apply path schedules so the 3s timer never
	// fires process.Kill on the test runner.
	t.Cleanup(func() {
		if m := rt.restart(); m != nil {
			m.cancelPending()
		}
	})
	return &IpCertificateService{
		Runtime:  rt,
		Settings: &SettingService{},
		acme:     issuer,
		now:      func() time.Time { return now },
	}
}

func TestIssueNowPersistsAndAppliesToPanel(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(160 * time.Hour)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, notAfter)
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "ACCOUNT-KEY-PEM"}
	svc := newIpCertTestService(t, issuer, now)

	status, err := svc.IssueNow(context.Background(), "93.184.216.34", "admin@example.com", 80, "panel", "")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Issued || status.TargetIP != "93.184.216.34" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if status.DaysRemaining < 6 || status.DaysRemaining > 7 {
		t.Fatalf("daysRemaining = %v, want ~6.7", status.DaysRemaining)
	}

	set := &SettingService{}
	certPath, _ := set.GetIpCertCertPath()
	keyPath, _ := set.GetIpCertKeyPath()
	if certPath == "" || keyPath == "" {
		t.Fatal("cert/key paths not persisted")
	}
	// Panel apply points the panel cert settings at the managed files.
	if webCert, _ := set.GetCertFile(); webCert != certPath {
		t.Fatalf("webCertFile = %q, want %q", webCert, certPath)
	}
	if webKey, _ := set.GetKeyFile(); webKey != keyPath {
		t.Fatalf("webKeyFile = %q, want %q", webKey, keyPath)
	}
	if na, _ := set.GetIpCertNotAfter(); na != notAfter.UTC().Format(time.RFC3339) {
		t.Fatalf("notAfter = %q, want %q", na, notAfter.UTC().Format(time.RFC3339))
	}

	// Internal account-key state must be stripped from the user-facing map.
	all, err := set.GetAllSetting()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range ipCertInternalSettingKeys {
		if _, ok := (*all)[k]; ok {
			t.Errorf("internal key %q leaked into GetAllSetting", k)
		}
	}
}

func TestIssueNowReusesAccountKey(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, now.Add(160*time.Hour))
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "ACCOUNT-KEY-PEM"}
	svc := newIpCertTestService(t, issuer, now)

	if _, err := svc.IssueNow(context.Background(), "93.184.216.34", "a@b.com", 80, "panel", ""); err != nil {
		t.Fatal(err)
	}
	if issuer.lastReq.AccountKeyPEM != "" {
		t.Fatal("first issue should start with no account key")
	}

	if _, err := svc.IssueNow(context.Background(), "93.184.216.34", "a@b.com", 80, "panel", ""); err != nil {
		t.Fatal(err)
	}
	if issuer.calls != 2 {
		t.Fatalf("issuer calls = %d, want 2", issuer.calls)
	}
	if issuer.lastReq.AccountKeyPEM != "ACCOUNT-KEY-PEM" {
		t.Fatalf("second issue account key = %q, want reused", issuer.lastReq.AccountKeyPEM)
	}
	if issuer.lastReq.RegistrationJSON == "" {
		t.Fatal("second issue should reuse the stored registration")
	}
}

func TestRenewIfNeededRespectsEnableAndThreshold(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, now.Add(160*time.Hour))
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "K"}
	svc := newIpCertTestService(t, issuer, now)
	set := &SettingService{}

	// Disabled → no-op even with a target set.
	mustSet(t, set, "ipCertTargetIP", "93.184.216.34")
	mustSet(t, set, "ipCertEmail", "a@b.com")
	renewed, err := svc.RenewIfNeeded(context.Background())
	if err != nil || renewed {
		t.Fatalf("disabled renew = (%v,%v), want (false,nil)", renewed, err)
	}

	// Enabled + fresh cert (far from expiry) → no renew.
	mustSet(t, set, "ipCertEnabled", "true")
	mustSet(t, set, "ipCertNotAfter", now.Add(120*time.Hour).Format(time.RFC3339))
	if renewed, err := svc.RenewIfNeeded(context.Background()); err != nil || renewed {
		t.Fatalf("fresh renew = (%v,%v), want (false,nil)", renewed, err)
	}

	// Enabled + near expiry → renews.
	mustSet(t, set, "ipCertNotAfter", now.Add(10*time.Hour).Format(time.RFC3339))
	if renewed, err := svc.RenewIfNeeded(context.Background()); err != nil || !renewed {
		t.Fatalf("near-expiry renew = (%v,%v), want (true,nil)", renewed, err)
	}
}

func TestRenewIfNeededForcesOnIpChange(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, now.Add(160*time.Hour))
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "K"}
	svc := newIpCertTestService(t, issuer, now)
	set := &SettingService{}

	// Enabled, cert still fresh (far from expiry), but the operator pointed
	// ipCertTargetIP at a different IP than the one last issued for. The stale
	// SAN must force a re-issue even though shouldRenew() alone would say no.
	mustSet(t, set, "ipCertEnabled", "true")
	mustSet(t, set, "ipCertTargetIP", "93.184.216.34")
	mustSet(t, set, "ipCertEmail", "a@b.com")
	mustSet(t, set, "ipCertApplyTarget", "panel")
	mustSet(t, set, "ipCertNotAfter", now.Add(120*time.Hour).Format(time.RFC3339))
	mustSet(t, set, "ipCertLastIP", "8.8.8.8")

	renewed, err := svc.RenewIfNeeded(context.Background())
	if err != nil || !renewed {
		t.Fatalf("ip-change renew = (%v,%v), want (true,nil)", renewed, err)
	}
	// After re-issue the recorded last IP converges on the configured target.
	if last, _ := set.getIpCertLastIP(); last != "93.184.216.34" {
		t.Fatalf("lastIP after renew = %q, want target IP", last)
	}

	// Same IP + fresh cert → no further renew (no spurious churn).
	if renewed, err := svc.RenewIfNeeded(context.Background()); err != nil || renewed {
		t.Fatalf("same-ip fresh renew = (%v,%v), want (false,nil)", renewed, err)
	}
}

func TestIssueNowReturnsApplyFailedButPersists(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(160 * time.Hour)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, notAfter)
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "K"}
	svc := newIpCertTestService(t, issuer, now)

	// Apply to a non-existent inbound TLS profile: issuance succeeds and is
	// persisted, but the apply step fails - IssueNow must surface the apply
	// error while keeping the stored state so a later retry can re-apply.
	status, err := svc.IssueNow(context.Background(), "93.184.216.34", "a@b.com", 80, "inbound:999999", "")
	if err == nil {
		t.Fatal("IssueNow with unresolvable inbound target = nil error, want apply failure")
	}
	if !strings.Contains(err.Error(), "apply failed") {
		t.Fatalf("error = %q, want 'apply failed'", err.Error())
	}
	// The certificate state is still persisted (the returned status reflects it).
	if !status.Issued {
		t.Fatal("status.Issued = false after issue-but-apply-failed, want true")
	}
	set := &SettingService{}
	if certPath, _ := set.GetIpCertCertPath(); certPath == "" {
		t.Fatal("cert path not persisted despite successful issuance")
	}
	if na, _ := set.GetIpCertNotAfter(); na != notAfter.UTC().Format(time.RFC3339) {
		t.Fatalf("notAfter = %q, want persisted %q", na, notAfter.UTC().Format(time.RFC3339))
	}
}

// TestIssueForCLIAppliesToPanelWithoutRuntime exercises the CLI issuance path:
// it must obtain, persist, and point the panel HTTPS cert settings at the new
// files WITHOUT a live Runtime (a one-shot CLI process has none) and WITHOUT
// scheduling a panel restart. A nil Runtime here would panic if the code path
// tried to restart, so this also asserts the no-restart contract.
func TestIssueForCLIAppliesToPanelWithoutRuntime(t *testing.T) {
	initSettingTestDB(t)
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(160 * time.Hour)
	certPEM, keyPEM := makeSelfSignedCertPEM(t, notAfter)
	issuer := &fakeIssuer{certPEM: certPEM, keyPEM: keyPEM, accountKeyOut: "ACCOUNT-KEY-PEM"}

	svc := &IpCertificateService{
		Settings: &SettingService{},
		acme:     issuer,
		now:      func() time.Time { return now },
		// Runtime intentionally nil: the CLI has no runtime and must not restart.
	}

	status, err := svc.IssueForCLI(context.Background(), "93.184.216.34", "admin@example.com", 80)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Issued || status.TargetIP != "93.184.216.34" {
		t.Fatalf("unexpected status: %+v", status)
	}

	set := &SettingService{}
	certPath, _ := set.GetIpCertCertPath()
	keyPath, _ := set.GetIpCertKeyPath()
	if certPath == "" || keyPath == "" {
		t.Fatal("cert/key paths not persisted")
	}
	if webCert, _ := set.GetCertFile(); webCert != certPath {
		t.Fatalf("webCertFile = %q, want %q", webCert, certPath)
	}
	if webKey, _ := set.GetKeyFile(); webKey != keyPath {
		t.Fatalf("webKeyFile = %q, want %q", webKey, keyPath)
	}
	// The CLI always targets the panel; the cron reads this back on renewal.
	if at, _ := set.GetIpCertApplyTarget(); at != "panel" {
		t.Fatalf("applyTarget = %q, want panel", at)
	}
}

func mustSet(t *testing.T, s *SettingService, key, value string) {
	t.Helper()
	if err := s.setString(key, value); err != nil {
		t.Fatalf("set %s: %v", key, err)
	}
}
