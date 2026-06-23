package service

import (
	"net/mail"
	"net/netip"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/ssrf"
)

// ipCertRenewThreshold is the remaining-validity window below which a managed
// IP certificate is re-issued. Let's Encrypt shortlived certs live ~160h
// (~6.7 days); renewing at <72h leaves several 12h cron passes of margin even
// if a renewal attempt fails, which is safer than 3x-ui's "renew at <6 days".
const ipCertRenewThreshold = 72 * time.Hour

// shouldRenew is the pure renewal decision. A zero notAfter means "never issued
// or unknown expiry" and always renews.
func shouldRenew(notAfter, now time.Time) bool {
	if notAfter.IsZero() {
		return true
	}
	return notAfter.Sub(now) < ipCertRenewThreshold
}

// ValidateIssuableIP is the exported wrapper around the package-private
// validator, kept for callers outside the service package.
func ValidateIssuableIP(raw string) error { return validateIssuableIP(raw) }

// validateIssuableIP rejects anything that is not a public IP literal Let's
// Encrypt could plausibly issue for. Reusing ssrf.IsBlockedAddr keeps the
// standalone HTTP-01 server from being pointed at private/loopback/link-local/
// multicast/CGNAT/metadata ranges. Shared by the settings validator, the API
// handler, and IssueNow (defense in depth). No DNS resolution is performed.
func validateIssuableIP(raw string) error {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return common.NewError("ip cert: target must be a valid IP literal")
	}
	if ssrf.IsBlockedAddr(addr) {
		return common.NewError("ip cert: IP is private/loopback/reserved and not issuable")
	}
	return nil
}

// validateIpCertEmail rejects malformed ACME-account emails. An empty value is
// allowed unless required (ACME permits account registration without a contact).
// net/mail.ParseAddress (stdlib) is used instead of a hand-rolled check so
// control characters, embedded newlines and degenerate forms like "@" are
// rejected; the parsed address must equal the input so display-name forms
// ("Name <a@b>") are not accepted. Shared by the settings validator and IssueNow.
func validateIpCertEmail(email string, required bool) error {
	email = strings.TrimSpace(email)
	if email == "" {
		if required {
			return common.NewError("ip cert: email is required when auto-renew is enabled")
		}
		return nil
	}
	if len(email) > 254 {
		return common.NewError("ip cert: email is too long")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return common.NewError("ip cert: email is not a valid address")
	}
	return nil
}

// validateIpCertPort bounds the HTTP-01 challenge port. Shared by the settings
// validator and IssueNow so a direct API issue cannot persist an out-of-range
// port that a later renewal would fail on.
func validateIpCertPort(port int) error {
	if port < 1 || port > 65535 {
		return common.NewError("ip cert: challenge port must be between 1 and 65535")
	}
	return nil
}

// --- typed settings accessors (user-editable) ---

func (s *SettingService) GetIpCertEnabled() (bool, error) {
	return s.getBool("ipCertEnabled")
}

// SetIpCertEnabled toggles cron auto-renewal. The CLI enables it on a successful
// issue ("issue and forget") and exposes an explicit disable.
func (s *SettingService) SetIpCertEnabled(v bool) error {
	value := "false"
	if v {
		value = "true"
	}
	return s.setString("ipCertEnabled", value)
}

func (s *SettingService) GetIpCertTargetIP() (string, error) {
	return s.getString("ipCertTargetIP")
}

func (s *SettingService) GetIpCertEmail() (string, error) {
	return s.getString("ipCertEmail")
}

func (s *SettingService) GetIpCertChallengePort() (int, error) {
	return s.getInt("ipCertChallengePort")
}

func (s *SettingService) GetIpCertApplyTarget() (string, error) {
	return s.getString("ipCertApplyTarget")
}

// Setters for the user-editable controls, written by IssueNow so a cert issued
// through a direct API call (without a prior settings save) still records the
// parameters the renewal cron needs.

func (s *SettingService) setIpCertTargetIP(v string) error {
	return s.setString("ipCertTargetIP", v)
}

func (s *SettingService) setIpCertEmail(v string) error {
	return s.setString("ipCertEmail", v)
}

func (s *SettingService) setIpCertChallengePort(v int) error {
	return s.setInt("ipCertChallengePort", v)
}

func (s *SettingService) setIpCertApplyTarget(v string) error {
	return s.setString("ipCertApplyTarget", v)
}

// --- typed settings accessors (machine-managed internal state) ---

func (s *SettingService) getIpCertAccountKey() (string, error) {
	return s.getString("ipCertAccountKey")
}

func (s *SettingService) setIpCertAccountKey(v string) error {
	return s.setEncryptedString("ipCertAccountKey", v)
}

func (s *SettingService) getIpCertAccountRegistration() (string, error) {
	return s.getString("ipCertAccountRegistration")
}

func (s *SettingService) setIpCertAccountRegistration(v string) error {
	return s.setString("ipCertAccountRegistration", v)
}

func (s *SettingService) getIpCertLastIP() (string, error) {
	return s.getString("ipCertLastIP")
}

func (s *SettingService) setIpCertLastIP(v string) error {
	return s.setString("ipCertLastIP", v)
}

func (s *SettingService) GetIpCertCertPath() (string, error) {
	return s.getString("ipCertCertPath")
}

func (s *SettingService) setIpCertCertPath(v string) error {
	return s.setString("ipCertCertPath", v)
}

func (s *SettingService) GetIpCertKeyPath() (string, error) {
	return s.getString("ipCertKeyPath")
}

func (s *SettingService) setIpCertKeyPath(v string) error {
	return s.setString("ipCertKeyPath", v)
}

func (s *SettingService) GetIpCertNotAfter() (string, error) {
	return s.getString("ipCertNotAfter")
}

func (s *SettingService) setIpCertNotAfter(v string) error {
	return s.setString("ipCertNotAfter", v)
}

func (s *SettingService) GetIpCertLastIssue() (string, error) {
	return s.getString("ipCertLastIssue")
}

func (s *SettingService) setIpCertLastIssue(v string) error {
	return s.setString("ipCertLastIssue", v)
}
