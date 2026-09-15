package migrateutil

import (
	"encoding/json"
	"fmt"

	"github.com/deposist/s-ui-x-extended/database/model"

	E "github.com/sagernet/sing/common/exceptions"
	"gorm.io/gorm"
)

// MigrateACMEToCertificateProvider rewrites the deprecated inline `acme` block
// of stored inbound TLS server configs (`tls` table, `server` column) into the
// 1.14 inline `certificate_provider` of type acme.
//
// Equivalence was confirmed against the 1.14 sources: every legacy
// InboundACMEOptions field maps 1:1 into ACMECertificateProviderOptions, and
// both the deprecated path (common/tls/acme.go startACME) and the new provider
// (service/acme/service.go NewCertificateProvider) build the same certmagic
// issuer with the same FileStorage data_directory, so certificate state and
// renewal behavior carry over unchanged.
//
// Scope guards:
//   - Only the core certmagic path is touched. The panel's own lego-based ACME
//     service (service/ip_certificate_*) writes certificate_path/key_path and
//     never sets `acme`, so it is left entirely alone (the two must not mix).
//   - If a TLS server block already has `certificate_provider`, the row is left
//     untouched: the core rejects acme+certificate_provider together, and the
//     already-migrated form must win.
//   - Idempotent: after conversion the `acme` key is gone, so reruns are no-ops.
func MigrateACMEToCertificateProvider(tx *gorm.DB) error {
	if tx == nil || !tx.Migrator().HasTable(&model.Tls{}) {
		return nil
	}
	var rows []model.Tls
	if err := tx.Model(&model.Tls{}).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		updated, changed, err := convertTLSACMEToCertificateProvider(row.Server, row.Name)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		if err := tx.Model(&model.Tls{}).Where("id = ?", row.Id).Update("server", updated).Error; err != nil {
			return E.Cause(err, "migrate acme to certificate_provider in tls ", fmt.Sprintf("%q", row.Name))
		}
	}
	return nil
}

// convertTLSACMEToCertificateProvider rewrites one TLS server blob. Returns the
// new blob and whether a change was made.
func convertTLSACMEToCertificateProvider(raw json.RawMessage, name string) (json.RawMessage, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, false, nil
	}
	var server map[string]json.RawMessage
	if err := json.Unmarshal(raw, &server); err != nil {
		return nil, false, E.Cause(err, "decode tls server ", fmt.Sprintf("%q", name))
	}
	acmeRaw, hasACME := server["acme"]
	if !hasACME {
		return raw, false, nil
	}
	// If certificate_provider is already present, the config is already in the
	// new form; drop the deprecated acme remnant but do not overwrite.
	if _, hasCP := server["certificate_provider"]; hasCP {
		delete(server, "acme")
		encoded, err := json.MarshalIndent(server, "", "  ")
		if err != nil {
			return nil, false, err
		}
		return encoded, true, nil
	}

	// Legacy acme must be an object to convert. An empty/null acme is simply
	// dropped (it produced no ACME behavior before either).
	var acme map[string]json.RawMessage
	if err := json.Unmarshal(acmeRaw, &acme); err != nil || acme == nil {
		delete(server, "acme")
		encoded, mErr := json.MarshalIndent(server, "", "  ")
		if mErr != nil {
			return nil, false, mErr
		}
		return encoded, true, nil
	}

	// Build {type: "acme", ...fields}. All legacy InboundACMEOptions fields map
	// 1:1 into ACMECertificateProviderOptions; fields the new schema added
	// (account_key, key_type, profile, http_client) have no legacy source and
	// stay absent.
	provider := map[string]json.RawMessage{"type": json.RawMessage(`"acme"`)}
	known := map[string]bool{
		"domain": true, "data_directory": true, "default_server_name": true,
		"email": true, "provider": true, "disable_http_challenge": true,
		"disable_tls_alpn_challenge": true, "alternative_http_port": true,
		"alternative_tls_port": true, "external_account": true, "dns01_challenge": true,
	}
	for key, value := range acme {
		if !known[key] {
			return nil, false, E.New("cannot migrate acme in tls ", fmt.Sprintf("%q", name),
				": unsupported legacy acme field ", fmt.Sprintf("%q", key), " with no certificate_provider equivalent")
		}
		provider[key] = value
	}
	encodedProvider, err := json.Marshal(provider)
	if err != nil {
		return nil, false, err
	}
	server["certificate_provider"] = encodedProvider
	delete(server, "acme")

	encoded, err := json.MarshalIndent(server, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return encoded, true, nil
}
