package importxui

import (
	"encoding/json"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util"

	"gorm.io/gorm"
)

// resolveLinkHostname returns the explicit hostname when set, otherwise falls
// back to the destination's configured sub/web domain. Centralizing the
// fallback here means no Apply/Import caller (API, CLI, scheduled sync) can
// reproduce the NULL-links bug by forgetting to pass a hostname.
func resolveLinkHostname(tx *gorm.DB, explicit string) string {
	if h := strings.TrimSpace(explicit); h != "" {
		return h
	}
	for _, key := range []string{"subDomain", "webDomain"} {
		var value string
		if err := tx.Model(&model.Setting{}).
			Where("key = ?", key).Limit(1).Pluck("value", &value).Error; err == nil {
			if v := strings.TrimSpace(value); v != "" {
				return v
			}
		}
	}
	return ""
}

// localLinkEntries generates the per-client "local" subscription links for an
// imported client, mirroring what the panel does on a normal client save
// (service.ClientService.updateLinksWithFixedInbounds). Without these the
// migrated client's Links column stays NULL, so none of its inbounds appear in
// the subscription or QR/Links view — the symptom of an imported inbound not
// being pulled into the subscription.
//
// hostname is the address baked into each link. When empty it returns nil, so a
// link with an empty host is never persisted.
func localLinkEntries(tx *gorm.DB, config json.RawMessage, inboundsJSON json.RawMessage, hostname string) ([]map[string]string, error) {
	if strings.TrimSpace(hostname) == "" {
		return nil, nil
	}
	var inboundIDs []uint
	if err := json.Unmarshal(inboundsJSON, &inboundIDs); err != nil {
		return nil, err
	}
	if len(inboundIDs) == 0 {
		return nil, nil
	}
	var inbounds []model.Inbound
	if err := tx.Model(model.Inbound{}).Preload("Tls").
		Where("id in ? and type in ?", inboundIDs, util.InboundTypeWithLink).
		Find(&inbounds).Error; err != nil {
		return nil, err
	}
	entries := make([]map[string]string, 0, len(inbounds))
	for i := range inbounds {
		for _, uri := range util.LinkGenerator(config, &inbounds[i], hostname) {
			entries = append(entries, map[string]string{
				"remark": inbounds[i].Tag,
				"type":   "local",
				"uri":    uri,
			})
		}
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return entries, nil
}

// buildClientLinks builds the marshaled Links column for a created/replaced
// client. Returns nil (Links left unset) when no link could be generated.
func buildClientLinks(tx *gorm.DB, config json.RawMessage, inboundsJSON json.RawMessage, hostname string) (json.RawMessage, error) {
	entries, err := localLinkEntries(tx, config, inboundsJSON, hostname)
	if err != nil || entries == nil {
		return nil, err
	}
	return json.MarshalIndent(entries, "", "  ")
}

// buildMergedClientLinks rebuilds the local links over the merged inbound set
// and preserves the existing client's non-local (external/sub) links, mirroring
// service.ClientService.updateLinksWithFixedInbounds. Returns nil when no local
// link was generated, so the caller leaves the existing Links untouched rather
// than clobbering them with NULL on a host-less import.
func buildMergedClientLinks(tx *gorm.DB, config json.RawMessage, inboundsJSON json.RawMessage, hostname string, existingLinks json.RawMessage) (json.RawMessage, error) {
	entries, err := localLinkEntries(tx, config, inboundsJSON, hostname)
	if err != nil || entries == nil {
		return nil, err
	}
	entries = append(entries, nonLocalLinks(existingLinks)...)
	return json.MarshalIndent(entries, "", "  ")
}

// nonLocalLinks returns the external/sub link entries from a client's stored
// Links, tolerating a NULL/empty/"null" column.
func nonLocalLinks(raw json.RawMessage) []map[string]string {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		return nil
	}
	kept := make([]map[string]string, 0, len(links))
	for _, link := range links {
		if link["type"] != "local" {
			kept = append(kept, link)
		}
	}
	return kept
}
