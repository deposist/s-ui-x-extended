package migrateutil

import (
	"encoding/json"
	"fmt"

	"github.com/deposist/s-ui-x-extended/database/model"

	E "github.com/sagernet/sing/common/exceptions"
	"gorm.io/gorm"
)

// MigrateDNS114 applies the sing-box 1.14 DNS deprecations to the stored
// `config` settings blob:
//
//   - dns.independent_cache is removed (1.14 always keys the cache by transport,
//     so the flag is a no-op; upstream: "simply remove the field").
//   - experimental.cache_file.store_rdrc is renamed to store_dns (upstream:
//     store_dns persists the full DNS cache and replaces store_rdrc).
//   - Legacy DNS address filters (ip_cidr / ip_is_private / ip_accept_any used
//     WITHOUT match_response) are converted to the evaluate + match_response
//     form, preserving rule order, the rule's own resolver, and the fallback.
//
// All transforms are idempotent. The address-filter conversion is the only one
// that can fail: it refuses to guess when the shape is ambiguous, and the whole
// transaction rolls back on error so stored config is never half-migrated.
func MigrateDNS114(tx *gorm.DB) error {
	if tx == nil || !tx.Migrator().HasTable(&model.Setting{}) {
		return nil
	}
	var row model.Setting
	if err := tx.Model(&model.Setting{}).Where("key = ?", "config").First(&row).Error; err != nil {
		// No config blob yet (fresh install) — nothing to migrate.
		return nil
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal([]byte(row.Value), &config); err != nil {
		return E.Cause(err, "decode config blob for DNS migration")
	}

	changed := false

	// experimental.cache_file.store_rdrc -> store_dns
	if expRaw, ok := config["experimental"]; ok {
		if newExp, did, err := migrateStoreRDRC(expRaw); err != nil {
			return err
		} else if did {
			config["experimental"] = newExp
			changed = true
		}
	}

	// dns: independent_cache removal + address-filter conversion
	if dnsRaw, ok := config["dns"]; ok {
		if newDNS, did, err := migrateDNSBlob(dnsRaw); err != nil {
			return err
		} else if did {
			config["dns"] = newDNS
			changed = true
		}
	}

	if !changed {
		return nil
	}
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return tx.Model(&model.Setting{}).Where("key = ?", "config").Update("value", string(encoded)).Error
}

// migrateStoreRDRC renames store_rdrc to store_dns inside experimental.cache_file.
// If store_dns is already set it wins and store_rdrc is dropped.
func migrateStoreRDRC(raw json.RawMessage) (json.RawMessage, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, false, nil
	}
	var exp map[string]json.RawMessage
	if err := json.Unmarshal(raw, &exp); err != nil {
		return nil, false, E.Cause(err, "decode experimental")
	}
	cfRaw, ok := exp["cache_file"]
	if !ok {
		return raw, false, nil
	}
	var cf map[string]json.RawMessage
	if err := json.Unmarshal(cfRaw, &cf); err != nil {
		return nil, false, E.Cause(err, "decode experimental.cache_file")
	}
	rdrc, has := cf["store_rdrc"]
	if !has {
		return raw, false, nil
	}
	// Only carry over when store_dns not already present (explicit new value wins).
	if _, hasNew := cf["store_dns"]; !hasNew {
		cf["store_dns"] = rdrc
	}
	delete(cf, "store_rdrc")
	// rdrc_timeout belonged to store_rdrc; store_dns manages its own expiry.
	delete(cf, "rdrc_timeout")
	encodedCF, err := json.Marshal(cf)
	if err != nil {
		return nil, false, err
	}
	exp["cache_file"] = encodedCF
	encoded, err := json.Marshal(exp)
	if err != nil {
		return nil, false, err
	}
	return encoded, true, nil
}

// migrateDNSBlob removes independent_cache and converts legacy address filters.
func migrateDNSBlob(raw json.RawMessage) (json.RawMessage, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, false, nil
	}
	var dns map[string]json.RawMessage
	if err := json.Unmarshal(raw, &dns); err != nil {
		return nil, false, E.Cause(err, "decode dns")
	}
	changed := false

	if _, ok := dns["independent_cache"]; ok {
		delete(dns, "independent_cache")
		changed = true
	}

	rulesRaw, ok := dns["rules"]
	if ok {
		newRules, did, err := convertDNSAddressFilterRules(rulesRaw)
		if err != nil {
			return nil, false, err
		}
		if did {
			dns["rules"] = newRules
			changed = true
		}
	}

	if !changed {
		return raw, false, nil
	}
	encoded, err := json.Marshal(dns)
	if err != nil {
		return nil, false, err
	}
	return encoded, true, nil
}

// addressFilterFields are the legacy response-filter match keys that require a
// preceding evaluate + match_response in 1.14.
var addressFilterFields = []string{"ip_cidr", "ip_is_private", "ip_accept_any"}

// convertDNSAddressFilterRules rewrites each legacy address-filter rule into an
// evaluate rule (same match conditions, same server) immediately followed by a
// match_response rule carrying the filter. Order is preserved; a rule that has
// no address filter is left untouched. This is intentionally NOT "insert
// evaluate before every rule": evaluate is only emitted where a response is
// actually needed for filtering.
func convertDNSAddressFilterRules(raw json.RawMessage) (json.RawMessage, bool, error) {
	var rules []json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, false, E.Cause(err, "decode dns.rules")
	}
	var out []json.RawMessage
	changed := false
	for i, ruleRaw := range rules {
		converted, did, err := convertDNSAddressFilterRule(ruleRaw, i)
		if err != nil {
			return nil, false, err
		}
		if did {
			changed = true
			out = append(out, converted...)
		} else {
			out = append(out, ruleRaw)
		}
	}
	if !changed {
		return raw, false, nil
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, false, err
	}
	return encoded, true, nil
}

// convertDNSAddressFilterRule converts one rule. Returns the replacement
// rule(s) — two rules (evaluate + match_response) when a conversion happened.
func convertDNSAddressFilterRule(raw json.RawMessage, index int) ([]json.RawMessage, bool, error) {
	var rule map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rule); err != nil {
		return nil, false, E.Cause(err, fmt.Sprintf("decode dns.rules[%d]", index))
	}

	// Logical rules (type: logical, mode and/or) nest sub-rules; the address
	// filter can appear on the logical rule's own action or inside. Handle only
	// the flat default-rule case here; logical rules with filters are rejected
	// with a precise reason rather than mis-converted.
	if t, ok := rule["type"]; ok {
		var ts string
		if json.Unmarshal(t, &ts) == nil && ts == "logical" {
			if ruleUsesAddressFilter(rule) {
				return nil, false, E.New(fmt.Sprintf(
					"dns rule[%d]: logical rule uses a legacy address filter (ip_cidr/ip_is_private/ip_accept_any); "+
						"convert it manually to an evaluate + match_response pair", index))
			}
			return []json.RawMessage{raw}, false, nil
		}
	}

	if !ruleUsesAddressFilter(rule) {
		return []json.RawMessage{raw}, false, nil
	}
	// Already in the new form (match_response present) — nothing to do.
	if _, ok := rule["match_response"]; ok {
		return []json.RawMessage{raw}, false, nil
	}
	// Only route actions resolve a response to filter. Other actions (reject,
	// predefined, route-options) don't produce a filterable answer.
	action := jsonString(rule, "action")
	if action != "" && action != "route" {
		return nil, false, E.New(fmt.Sprintf(
			"dns rule[%d]: legacy address filter with action %q cannot be auto-converted; convert manually", index, action))
	}
	server := jsonString(rule, "server")
	if server == "" {
		return nil, false, E.New(fmt.Sprintf(
			"dns rule[%d]: legacy address filter without a server cannot be auto-converted; add an evaluate server", index))
	}

	// Split the rule's fields into query-match conditions (shared by both the
	// evaluate and the match_response rule) and the response filter (moved to
	// the match_response rule only).
	evaluate := map[string]json.RawMessage{}
	filterRule := map[string]json.RawMessage{}
	// action-specific keys that belong to the route action on the filter rule.
	actionKeys := map[string]bool{"strategy": true, "disable_cache": true, "rewrite_ttl": true, "client_subnet": true}
	for key, value := range rule {
		switch {
		case key == "action":
			// set explicitly below
		case key == "server":
			// set explicitly below
		case containsString(addressFilterFields, key):
			filterRule[key] = value
		case actionKeys[key]:
			// route action options stay on the final route rule
			filterRule[key] = value
		default:
			// query match conditions shared by both rules
			evaluate[key] = value
			filterRule[key] = value
		}
	}

	evaluate["action"] = json.RawMessage(`"evaluate"`)
	evaluate["server"] = jsonStringRaw(server)

	filterRule["match_response"] = json.RawMessage("true")
	filterRule["action"] = json.RawMessage(`"route"`)
	filterRule["server"] = jsonStringRaw(server)

	encodedEval, err := json.Marshal(evaluate)
	if err != nil {
		return nil, false, err
	}
	encodedFilter, err := json.Marshal(filterRule)
	if err != nil {
		return nil, false, err
	}
	return []json.RawMessage{encodedEval, encodedFilter}, true, nil
}

// ruleUsesAddressFilter reports whether a rule uses a legacy address filter
// field (which in 1.14 requires match_response).
func ruleUsesAddressFilter(rule map[string]json.RawMessage) bool {
	for _, field := range addressFilterFields {
		if _, ok := rule[field]; ok {
			return true
		}
	}
	return false
}

func jsonString(rule map[string]json.RawMessage, key string) string {
	raw, ok := rule[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func jsonStringRaw(s string) json.RawMessage {
	encoded, _ := json.Marshal(s)
	return encoded
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
