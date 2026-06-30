package migrateutil

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

var legacyInboundRuleActionFields = map[string]struct{}{
	"sniff":                        {},
	"sniff_override_destination":   {},
	"sniff_timeout":                {},
	"domain_strategy":              {},
	"udp_disable_domain_unmapping": {},
}

const defaultLegacyInboundConfig = `{
  "log": {
    "level": "info"
  },
  "dns": {
    "servers": [],
    "rules": []
  },
  "route": {
    "rules": [
      {
        "action": "sniff"
      },
      {
        "protocol": [
          "dns"
        ],
        "action": "hijack-dns"
      }
    ]
  },
  "experimental": {}
}`

type legacyInboundRuleActionSnapshot struct {
	ID                        uint
	Tag                       string
	Sniff                     bool
	SniffTimeout              string
	SniffOverrideDestination  bool
	DomainStrategy            string
	UDPDisableDomainUnmapping bool
}

type legacyInboundRuleActionRow struct {
	inbound  model.Inbound
	snapshot legacyInboundRuleActionSnapshot
	options  map[string]any
}

// MigrateLegacyInboundRuleActionFields moves sing-box 1.11+ legacy inbound
// rule-action fields out of inbounds.options and into route.rules, then removes
// the deprecated inbound keys. It is intentionally idempotent: rerunning it
// does not create duplicate equivalent route rules.
func MigrateLegacyInboundRuleActionFields(tx *gorm.DB) error {
	if tx == nil {
		return nil
	}
	if !tx.Migrator().HasTable(&model.Inbound{}) {
		return nil
	}

	rows, err := legacyInboundRuleActionRows(tx)
	if err != nil || len(rows) == 0 {
		return err
	}

	if tx.Migrator().HasTable(&model.Setting{}) {
		if err := migrateLegacyInboundRouteRules(tx, rows); err != nil {
			return err
		}
	}

	for _, row := range rows {
		cleaned, err := json.MarshalIndent(row.options, "", "  ")
		if err != nil {
			return err
		}
		if err := tx.Model(&model.Inbound{}).Where("id = ?", row.inbound.Id).Update("options", json.RawMessage(cleaned)).Error; err != nil {
			return err
		}
	}
	return nil
}

func legacyInboundRuleActionRows(tx *gorm.DB) ([]legacyInboundRuleActionRow, error) {
	var inbounds []model.Inbound
	if err := tx.Model(&model.Inbound{}).Find(&inbounds).Error; err != nil {
		return nil, err
	}
	rows := make([]legacyInboundRuleActionRow, 0)
	for _, inbound := range inbounds {
		options, err := decodeLegacyInboundOptions(inbound.Options)
		if err != nil {
			return nil, err
		}
		if len(options) == 0 || !hasLegacyInboundRuleActionField(options) {
			continue
		}
		snapshot := legacyInboundRuleActionSnapshot{ID: inbound.Id, Tag: inbound.Tag}
		snapshot.Sniff, _ = boolOption(options, "sniff")
		snapshot.SniffTimeout, _ = stringOption(options, "sniff_timeout")
		snapshot.SniffOverrideDestination, _ = boolOption(options, "sniff_override_destination")
		snapshot.DomainStrategy, _ = stringOption(options, "domain_strategy")
		snapshot.UDPDisableDomainUnmapping, _ = boolOption(options, "udp_disable_domain_unmapping")
		for key := range legacyInboundRuleActionFields {
			delete(options, key)
		}
		rows = append(rows, legacyInboundRuleActionRow{inbound: inbound, snapshot: snapshot, options: options})
	}
	return rows, nil
}

func decodeLegacyInboundOptions(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, nil
	}
	var options map[string]any
	if err := json.Unmarshal(raw, &options); err != nil {
		return nil, err
	}
	if options == nil {
		options = map[string]any{}
	}
	return options, nil
}

func hasLegacyInboundRuleActionField(options map[string]any) bool {
	for key := range legacyInboundRuleActionFields {
		if _, ok := options[key]; ok {
			return true
		}
	}
	return false
}

func migrateLegacyInboundRouteRules(tx *gorm.DB, rows []legacyInboundRuleActionRow) error {
	config, configRowExists, err := legacyInboundConfig(tx)
	if err != nil {
		return err
	}
	route := ensureObject(config, "route")
	rules := ensureArray(route, "rules")
	prependRules := make([]any, 0)
	for _, row := range rows {
		snapshot := row.snapshot
		if snapshot.Tag == "" {
			continue
		}
		if snapshot.DomainStrategy != "" {
			rule := map[string]any{
				"inbound":  []any{snapshot.Tag},
				"action":   "resolve",
				"strategy": snapshot.DomainStrategy,
			}
			if !routeRuleExists(rules, rule) && !routeRuleExists(prependRules, rule) {
				prependRules = append(prependRules, rule)
			}
		}
		if snapshot.Sniff {
			rule := map[string]any{
				"inbound": []any{snapshot.Tag},
				"action":  "sniff",
			}
			if snapshot.SniffTimeout != "" {
				rule["timeout"] = snapshot.SniffTimeout
			}
			if !routeRuleExists(rules, rule) && !routeRuleExists(prependRules, rule) {
				prependRules = append(prependRules, rule)
			}
		}
		if snapshot.UDPDisableDomainUnmapping {
			rule := map[string]any{
				"inbound":                      []any{snapshot.Tag},
				"action":                       "route-options",
				"udp_disable_domain_unmapping": true,
			}
			if !routeRuleExists(rules, rule) && !routeRuleExists(prependRules, rule) {
				prependRules = append(prependRules, rule)
			}
		}
	}
	if len(prependRules) == 0 {
		return nil
	}
	route["rules"] = append(prependRules, rules...)
	config["route"] = route
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if configRowExists {
		return tx.Model(&model.Setting{}).Where("key = ?", "config").Update("value", string(encoded)).Error
	}
	return tx.Create(&model.Setting{Key: "config", Value: string(encoded)}).Error
}

func legacyInboundConfig(tx *gorm.DB) (map[string]any, bool, error) {
	var setting model.Setting
	err := tx.Model(&model.Setting{}).Where("key = ?", "config").First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		config, parseErr := defaultLegacyConfigMap()
		return config, false, parseErr
	}
	if err != nil {
		return nil, false, err
	}
	if setting.Value == "" {
		config, parseErr := defaultLegacyConfigMap()
		return config, true, parseErr
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(setting.Value), &config); err != nil {
		return nil, false, err
	}
	if config == nil {
		config = map[string]any{}
	}
	return config, true, nil
}

func defaultLegacyConfigMap() (map[string]any, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(defaultLegacyInboundConfig), &config); err != nil {
		return nil, err
	}
	return config, nil
}

func ensureObject(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	object := map[string]any{}
	parent[key] = object
	return object
}

func ensureArray(parent map[string]any, key string) []any {
	if value, ok := parent[key].([]any); ok {
		return value
	}
	array := []any{}
	parent[key] = array
	return array
}

func routeRuleExists(existing []any, candidate map[string]any) bool {
	for _, item := range existing {
		rule, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if routeRulesEquivalent(rule, candidate) {
			return true
		}
	}
	return false
}

func routeRulesEquivalent(existing map[string]any, candidate map[string]any) bool {
	if action, _ := existing["action"].(string); action != candidate["action"] {
		return false
	}
	if !sameStringSet(existing["inbound"], candidate["inbound"]) {
		return false
	}
	if !sameRouteRulePayload(existing, candidate) {
		return false
	}
	return true
}

func sameRouteRulePayload(existing map[string]any, candidate map[string]any) bool {
	for key, value := range candidate {
		if key == "action" || key == "inbound" {
			continue
		}
		if !reflect.DeepEqual(existing[key], value) {
			return false
		}
	}
	for key, value := range existing {
		if key == "action" || key == "inbound" {
			continue
		}
		candidateValue, ok := candidate[key]
		if !ok || !reflect.DeepEqual(value, candidateValue) {
			return false
		}
	}
	return true
}

func sameStringSet(left any, right any) bool {
	leftSet := stringSet(left)
	rightSet := stringSet(right)
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if !rightSet[value] {
			return false
		}
	}
	return true
}

func stringSet(value any) map[string]bool {
	out := map[string]bool{}
	switch typed := value.(type) {
	case string:
		if typed != "" {
			out[typed] = true
		}
	case []any:
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out[s] = true
			}
		}
	case []string:
		for _, s := range typed {
			if s != "" {
				out[s] = true
			}
		}
	}
	return out
}

func boolOption(options map[string]any, key string) (bool, bool) {
	value, ok := options[key]
	if !ok {
		return false, false
	}
	if typed, ok := value.(bool); ok {
		return typed, true
	}
	return false, true
}

func stringOption(options map[string]any, key string) (string, bool) {
	value, ok := options[key]
	if !ok {
		return "", false
	}
	if typed, ok := value.(string); ok {
		return typed, true
	}
	return "", true
}
