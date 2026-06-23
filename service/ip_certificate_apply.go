package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// ipCertPanelRestartDelay gives the HTTP response time to flush before the
// panel restarts to pick up the new certificate.
const ipCertPanelRestartDelay = 3 * time.Second

// applyToTarget installs the freshly issued cert/key at the requested target:
// either the panel HTTPS listener (settings + restart) or an inbound TLS
// profile (hot-reload, no core restart).
func (s *IpCertificateService) applyToTarget(target, certPath, keyPath, hostname string) error {
	if target == "" || target == "panel" {
		return s.applyToPanel(certPath, keyPath)
	}
	rest, ok := strings.CutPrefix(target, "inbound:")
	if !ok {
		return common.NewError("ip cert: unknown apply target: ", target)
	}
	tlsID, err := strconv.Atoi(rest)
	if err != nil || tlsID <= 0 {
		return common.NewError("ip cert: invalid inbound tls id: ", rest)
	}
	return s.applyToInboundTls(uint(tlsID), certPath, keyPath, hostname)
}

// applyToPanel points the panel cert settings at the managed files and
// schedules a panel restart so web.go reloads them. Used from inside the live
// panel (the renewal cron); the CLI path uses setPanelCertSettings without the
// restart.
func (s *IpCertificateService) applyToPanel(certPath, keyPath string) error {
	if err := s.setPanelCertSettings(certPath, keyPath); err != nil {
		return err
	}
	// The cert settings are now written; the panel only re-reads them on a full
	// restart. Use a BLOCKING restart so a collision with another in-flight
	// operation cannot silently drop this restart (which would leave the live
	// listener serving the old certificate while the renewal reports success and
	// advances ipCertNotAfter, suppressing the next retry).
	if manager := runtimeOrDefault(s.Runtime).restart(); manager != nil {
		return manager.scheduleRestartBlocking(ipCertPanelRestartDelay)
	}
	panel := &PanelService{Runtime: s.Runtime}
	return panel.RestartPanel(ipCertPanelRestartDelay)
}

// setPanelCertSettings points the panel HTTPS cert settings at the managed
// files without restarting anything. The panel re-reads webCertFile/webKeyFile
// only on a full restart, so the caller is responsible for restarting the panel.
func (s *IpCertificateService) setPanelCertSettings(certPath, keyPath string) error {
	set := s.settings()
	if err := set.setString("webCertFile", certPath); err != nil {
		return err
	}
	return set.setString("webKeyFile", keyPath)
}

// applyToInboundTls patches the chosen TLS row's server block with the managed
// certificate_path/key_path and routes the change through ConfigService.Save so
// only the affected inbounds/services hot-reload.
func (s *IpCertificateService) applyToInboundTls(tlsID uint, certPath, keyPath, hostname string) error {
	db := database.GetDB()
	if db == nil {
		return common.NewError("ip cert: database is not initialized")
	}
	var tls model.Tls
	if err := db.Model(model.Tls{}).Where("id = ?", tlsID).First(&tls).Error; err != nil {
		return common.NewError("ip cert: tls profile not found: ", err.Error())
	}

	patchedServer, err := patchTlsServerBlock(tls.Server, certPath, keyPath)
	if err != nil {
		return err
	}
	tls.Server = patchedServer

	payload, err := json.Marshal(tls)
	if err != nil {
		return err
	}

	config := NewConfigServiceWithRuntime(s.Runtime)
	_, err = config.Save("tls", "edit", payload, "", "", hostname)
	return err
}

// patchTlsServerBlock points a sing-box TLS server block at the managed
// certificate/key files. Pure (no I/O): it parses the existing server JSON,
// sets certificate_path/key_path, and drops any inline certificate/key bytes
// that would otherwise shadow the file paths in sing-box.
func patchTlsServerBlock(serverJSON json.RawMessage, certPath, keyPath string) (json.RawMessage, error) {
	server := map[string]json.RawMessage{}
	if len(serverJSON) > 0 {
		if err := json.Unmarshal(serverJSON, &server); err != nil {
			return nil, common.NewError("ip cert: tls server block is not an object: ", err.Error())
		}
	}
	if err := setJSONStringField(server, "certificate_path", certPath); err != nil {
		return nil, err
	}
	if err := setJSONStringField(server, "key_path", keyPath); err != nil {
		return nil, err
	}
	delete(server, "certificate")
	delete(server, "key")
	return json.Marshal(server)
}

func setJSONStringField(obj map[string]json.RawMessage, key, value string) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	obj[key] = encoded
	return nil
}
