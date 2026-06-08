package importxui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/gorm"
)

func markOnlyNew(plan *MigrationPlan) {
	for i := range plan.Items {
		if plan.Items[i].Conflict {
			plan.Items[i].Action = ActionSkip
		}
	}
}

func planHistorical(ctx context.Context, src *sourceDB, plan *MigrationPlan) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	clients, err := src.dialect.ReadClients(src.sqlDB())
	if err != nil {
		return err
	}
	outbounds, err := src.outboundTraffics()
	if err != nil {
		return err
	}
	count := 0
	for _, row := range clients {
		if row.Email != "" && (row.Up > 0 || row.Down > 0) {
			count++
		}
	}
	for _, row := range outbounds {
		if row.Tag != "" && (row.Up > 0 || row.Down > 0) {
			count++
		}
	}
	preview, err := marshalJSON(map[string]any{
		"client_traffics":   len(clients),
		"outbound_traffics": len(outbounds),
		"mode":              "aggregated_only",
	})
	if err != nil {
		return err
	}
	plan.Items = append(plan.Items, PlanItem{
		Kind:        KindHistory,
		SrcID:       "traffic",
		SrcTag:      "client_traffics/outbound_traffics",
		DstTag:      "stats",
		Action:      ActionCreate,
		PreviewJSON: preview,
		Warnings:    []string{"historical_aggregated_only"},
	})
	plan.Defaults.IncludeHistory = count > 0
	return nil
}

func (s *applyState) applyHistorical(ctx context.Context, tx *gorm.DB, src *sourceDB, opts ApplyOptions) error {
	if !s.hasKind(KindHistory) {
		return nil
	}
	item := s.item(KindHistory, "traffic")
	if item.Action == ActionSkip {
		return nil
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	now := time.Now().Unix()
	if opts.Now != nil {
		now = opts.Now()
	}
	var stats []model.Stats
	clients, err := src.dialect.ReadClients(src.sqlDB())
	if err != nil {
		return err
	}
	for _, row := range clients {
		if row.Email == "" {
			continue
		}
		if row.Up > 0 {
			stats = append(stats, model.Stats{DateTime: now, Resource: "client", Tag: row.Email, Direction: true, Traffic: row.Up})
		}
		if row.Down > 0 {
			stats = append(stats, model.Stats{DateTime: now, Resource: "client", Tag: row.Email, Direction: false, Traffic: row.Down})
		}
	}
	outbounds, err := src.outboundTraffics()
	if err != nil {
		return err
	}
	for _, row := range outbounds {
		if row.Tag == "" {
			continue
		}
		if row.Up > 0 {
			stats = append(stats, model.Stats{DateTime: now, Resource: "outbound", Tag: row.Tag, Direction: true, Traffic: row.Up})
		}
		if row.Down > 0 {
			stats = append(stats, model.Stats{DateTime: now, Resource: "outbound", Tag: row.Tag, Direction: false, Traffic: row.Down})
		}
	}
	if len(stats) > 0 {
		if err := database.CreateInBatchesSafe(tx, &stats); err != nil {
			return err
		}
	}
	s.report.Summary.Historical.Total = len(stats)
	s.report.Summary.Historical.Imported = len(stats)
	s.report.warn("historical_aggregated_only")
	s.progress("historical", "stats")
	return nil
}

// createNewEndpoints persists WARP/wireguard-outbound endpoints, creating each
// only when no endpoint with that tag already exists. It never overwrites an
// existing endpoint, so re-imports and scheduled sync stay idempotent and a
// user-tuned (or same-tagged) endpoint — including its private key — is left
// untouched. The routing rule still references the tag, which now exists either
// way, so there is no dangling reference.
func createNewEndpoints(tx *gorm.DB, endpoints []model.Endpoint, report *Report) error {
	for i := range endpoints {
		ep := &endpoints[i]
		var existing model.Endpoint
		err := tx.Where("tag = ?", ep.Tag).First(&existing).Error
		if err != nil && !database.IsNotFound(err) {
			return err
		}
		if err == nil {
			report.Summary.Endpoints.Skipped++
			report.warn(fmt.Sprintf("endpoint %q already exists; WARP outbound left unchanged", ep.Tag))
			continue
		}
		if err := tx.Create(ep).Error; err != nil {
			return err
		}
		report.Summary.Endpoints.Imported++
		report.warn(fmt.Sprintf("imported WARP endpoint %q from xray wireguard outbound", ep.Tag))
	}
	return nil
}

// createNewOutbounds persists proxy outbounds (vmess/vless/trojan/shadowsocks/
// socks/http) mapped from the source Xray outbounds, creating each only when no
// outbound with that tag already exists. Like createNewEndpoints it never
// overwrites an existing outbound, so re-imports and scheduled sync stay
// idempotent and an operator-tuned (or same-tagged) outbound is left untouched;
// the routing rule still references the tag, which exists either way.
func createNewOutbounds(tx *gorm.DB, outbounds []model.Outbound, report *Report) error {
	for i := range outbounds {
		ob := &outbounds[i]
		var existing model.Outbound
		err := tx.Where("tag = ?", ob.Tag).First(&existing).Error
		if err != nil && !database.IsNotFound(err) {
			return err
		}
		if err == nil {
			report.Summary.Outbounds.Skipped++
			report.warn(fmt.Sprintf("outbound %q already exists; left unchanged", ob.Tag))
			continue
		}
		if err := tx.Create(ob).Error; err != nil {
			return err
		}
		report.Summary.Outbounds.Imported++
		report.warn(fmt.Sprintf("imported %s outbound %q from xray outbound", ob.Type, ob.Tag))
	}
	return nil
}

// ensureDirectOutbound appends a built-in direct outbound (tag "direct") to
// outbounds when the migrated routing references "direct" — either a rule that
// routes to it (from an Xray freedom outbound) or a remote rule set whose
// download_detour is "direct" — but none exists yet. sing-box only auto-creates
// a fallback direct outbound when there are zero outbounds, so a migrated config
// that has any other outbound would otherwise fail at route time with
// "outbound not found: direct". It is a no-op when a direct outbound already
// exists in the slice or in the DB (the s-ui default seeded by InitDB), so a
// normal import never makes createNewOutbounds report a misleading
// "outbound \"direct\" already exists" skip.
func ensureDirectOutbound(tx *gorm.DB, outbounds []model.Outbound, mapped map[string]any) []model.Outbound {
	if !routingReferencesDirect(mapped) {
		return outbounds
	}
	for i := range outbounds {
		if outbounds[i].Tag == directOutboundTag {
			return outbounds
		}
	}
	var existing model.Outbound
	if err := tx.Where("tag = ?", directOutboundTag).First(&existing).Error; err == nil {
		return outbounds // already in the DB (e.g. the InitDB-seeded default)
	}
	return append(outbounds, model.Outbound{Type: directOutboundTag, Tag: directOutboundTag})
}

// mappedHasDNS reports whether the migrated config carries DNS servers or rules.
// A source whose only migratable content is DNS (no routing rules, no proxy
// outbounds, no endpoints) must not be skipped, or its DNS is silently dropped.
func mappedHasDNS(mapped map[string]any) bool {
	dns, ok := mapped["dns"].(map[string]any)
	if !ok {
		return false
	}
	return len(toAnySlice(dns["servers"])) > 0 || len(toAnySlice(dns["rules"])) > 0
}

// routingReferencesDirect reports whether any migrated route rule routes to the
// direct outbound or any migrated remote rule set downloads through it.
func routingReferencesDirect(mapped map[string]any) bool {
	route, ok := mapped["route"].(map[string]any)
	if !ok {
		return false
	}
	for _, r := range toAnySlice(route["rules"]) {
		if m, ok := r.(map[string]any); ok {
			if ob, _ := m["outbound"].(string); ob == directOutboundTag {
				return true
			}
		}
	}
	for _, rs := range toAnySlice(route["rule_set"]) {
		if m, ok := rs.(map[string]any); ok {
			if d, _ := m["download_detour"].(string); d == directOutboundTag {
				return true
			}
		}
	}
	return false
}

// planRoutingDisabledNotice surfaces a single warning-only plan item when
// routing import is turned off but the source Xray config still contains proxy
// outbounds or WARP endpoints. Those live in the same xrayConfig and are only
// migrated as part of routing import (an outbound is useless without the rules
// that reference it), so without this notice they would vanish from the
// migration with no plan item and no warning — the exact silent-loss the
// operator hit before this feature existed.
func planRoutingDisabledNotice(ctx context.Context, src *sourceDB, plan *MigrationPlan) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	xrayConfig, err := src.xrayConfig()
	if err != nil {
		return err
	}
	endpoints, outbounds, _, _ := mapXrayOutbounds(xrayConfig)
	if len(endpoints) == 0 && len(outbounds) == 0 {
		return nil
	}
	plan.Items = append(plan.Items, warningOnlyItem(
		KindRouting, "xrayConfig", "xrayConfig.outbounds", "config",
		[]string{fmt.Sprintf("%d proxy outbound(s) and %d WARP endpoint(s) in the source are not migrated because routing import is disabled; enable \"Include routing\" to migrate them", len(outbounds), len(endpoints))},
	))
	return nil
}

func planRouting(ctx context.Context, src *sourceDB, plan *MigrationPlan) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	xrayConfig, err := src.xrayConfig()
	if err != nil {
		return err
	}
	endpoints, outbounds, targets, outboundWarnings := mapXrayOutbounds(xrayConfig)
	mapped, warnings, mappedCount, manualCount := MapXrayRouting(xrayConfig, targets)
	warnings = append(outboundWarnings, warnings...)
	preview, err := marshalJSON(mapped)
	if err != nil {
		return err
	}
	action := ActionCreate
	if xrayConfig == "" || (mappedCount == 0 && manualCount == 0 && len(endpoints) == 0 && len(outbounds) == 0 && !mappedHasDNS(mapped)) {
		action = ActionSkip
	}
	plan.Items = append(plan.Items, PlanItem{
		Kind:        KindRouting,
		SrcID:       "xrayConfig",
		SrcTag:      "xrayConfig.routing",
		DstTag:      "config",
		Action:      action,
		PreviewJSON: preview,
		Warnings:    warnings,
	})
	return nil
}

func (s *applyState) applyRouting(ctx context.Context, tx *gorm.DB, src *sourceDB, _ ApplyOptions) error {
	if !s.hasKind(KindRouting) {
		return nil
	}
	item := s.item(KindRouting, "xrayConfig")
	if item.Action == ActionSkip {
		return nil
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	xrayConfig, err := src.xrayConfig()
	if err != nil {
		return err
	}
	// WARP (and any wireguard outbound) becomes an s-ui endpoint and proxy
	// outbounds become s-ui outbounds; create those first so the routing rules
	// below can target them by tag, then map the rules. blackhole/freedom/dns
	// outbounds resolve to a reject action / the direct outbound / hijack-dns.
	endpoints, outbounds, targets, outboundWarnings := mapXrayOutbounds(xrayConfig)
	s.report.warnAll(outboundWarnings)
	mapped, warnings, mappedCount, manualCount := MapXrayRouting(xrayConfig, targets)
	// A migrated rule that routes to "direct" — or a remote rule set that
	// downloads via download_detour:"direct" — needs a real direct outbound.
	// sing-box 1.11+ only auto-creates one when there are no outbounds and no
	// route.final, so a config with any other outbound would otherwise fail at
	// route time with "outbound not found: direct".
	outbounds = ensureDirectOutbound(tx, outbounds, mapped)
	if err := createNewEndpoints(tx, endpoints, s.report); err != nil {
		return err
	}
	for i := range endpoints {
		s.progress("endpoints", endpoints[i].Tag)
	}
	if err := createNewOutbounds(tx, outbounds, s.report); err != nil {
		return err
	}
	for i := range outbounds {
		s.progress("outbounds", outbounds[i].Tag)
	}
	if err := mergeRoutingIntoConfig(tx, mapped); err != nil {
		return err
	}
	s.report.Summary.Routing.Total = mappedCount + manualCount
	s.report.Summary.Routing.Imported = mappedCount
	s.report.Summary.Routing.Skipped = manualCount
	s.report.warnAll(warnings)
	s.progress("routing", "config")
	return nil
}

// defaultLiveConfig mirrors service.defaultConfig — the baseline sing-box config
// the panel falls back to before any config is saved. It is duplicated here
// (rather than importing the service layer from a database subpackage) so the
// routing merge can seed it when no `config` row exists, keeping the default
// sniff/hijack-dns rules. Keep the route/dns skeleton in sync with
// service.defaultConfig; the merge only appends, so a stale copy cannot drop a
// migrated rule.
const defaultLiveConfig = `{
  "log": {"level": "info"},
  "dns": {"servers": [], "rules": []},
  "route": {"rules": [{"action": "sniff"}, {"protocol": ["dns"], "action": "hijack-dns"}]},
  "experimental": {}
}`

// mergeRoutingIntoConfig merges the migrated route rules / rule sets and DNS
// servers/rules into the live sing-box config (the "config" setting that
// ConfigService.GetConfig loads), rather than a side setting nothing reads.
// Existing rules are preserved: migrated rules are appended after them and rule
// sets / DNS servers are de-duplicated by tag, so a re-import stays idempotent
// and an operator's own routing is never clobbered.
func mergeRoutingIntoConfig(tx *gorm.DB, mapped map[string]any) error {
	var current string
	if err := tx.Model(model.Setting{}).Select("value").Where("key = ?", "config").Scan(&current).Error; err != nil {
		return err
	}
	if strings.TrimSpace(current) == "" {
		// The panel falls back to a default config until one is saved; that
		// default is not stored as a row. Seed it here so writing the `config`
		// row keeps the default sniff/hijack-dns rules and dns skeleton instead
		// of shadowing them with a route that has only the migrated rules.
		current = defaultLiveConfig
	}
	cfg := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(current), &cfg); err != nil {
		return fmt.Errorf("routing: existing config is not valid JSON: %w", err)
	}
	changed := false

	if route, ok := mapped["route"].(map[string]any); ok {
		newRules := toAnySlice(route["rules"])
		newRuleSets := toAnySlice(route["rule_set"])
		if len(newRules) > 0 || len(newRuleSets) > 0 {
			dst := decodeConfigObject(cfg["route"])
			if len(newRules) > 0 {
				dst["rules"] = appendUniqueRules(toAnySlice(dst["rules"]), newRules)
			}
			if len(newRuleSets) > 0 {
				dst["rule_set"] = mergeByTag(toAnySlice(dst["rule_set"]), newRuleSets)
			}
			enc, err := json.Marshal(dst)
			if err != nil {
				return err
			}
			cfg["route"] = enc
			changed = true
		}
	}

	if dns, ok := mapped["dns"].(map[string]any); ok && len(dns) > 0 {
		newServers := toAnySlice(dns["servers"])
		newDNSRules := toAnySlice(dns["rules"])
		if len(newServers) > 0 || len(newDNSRules) > 0 {
			dst := decodeConfigObject(cfg["dns"])
			if len(newServers) > 0 {
				dst["servers"] = mergeByTag(toAnySlice(dst["servers"]), newServers)
			}
			if len(newDNSRules) > 0 {
				dst["rules"] = appendUniqueRules(toAnySlice(dst["rules"]), newDNSRules)
			}
			// Carry top-level DNS knobs only when the live config has not set them.
			for _, k := range []string{"strategy", "final", "client_subnet"} {
				if v, ok := dns[k]; ok {
					if _, exists := dst[k]; !exists {
						dst[k] = v
					}
				}
			}
			enc, err := json.Marshal(dst)
			if err != nil {
				return err
			}
			cfg["dns"] = enc
			changed = true
		}
	}

	if !changed {
		return nil
	}
	merged, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return upsertSetting(tx, "config", string(merged))
}

// decodeConfigObject decodes a config sub-object (route/dns) into a map, or an
// empty map when absent.
func decodeConfigObject(raw json.RawMessage) map[string]any {
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

// toAnySlice coerces a value to []any (nil when it is not a slice).
func toAnySlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// appendUniqueRules appends rules to existing, skipping any whose canonical JSON
// already appears. Route/DNS rules have no tag to key on, so without this a
// re-import or scheduled sync would append the same migrated rules every run and
// grow the live config without bound. Map keys marshal in sorted order, so an
// existing rule read back from the DB and a freshly built identical rule produce
// the same bytes.
func appendUniqueRules(existing, additions []any) []any {
	seen := map[string]struct{}{}
	for _, e := range existing {
		if key, err := json.Marshal(e); err == nil {
			seen[string(key)] = struct{}{}
		}
	}
	for _, a := range additions {
		key, err := json.Marshal(a)
		if err != nil {
			existing = append(existing, a)
			continue
		}
		if _, dup := seen[string(key)]; dup {
			continue
		}
		seen[string(key)] = struct{}{}
		existing = append(existing, a)
	}
	return existing
}

// mergeByTag appends additions to existing, skipping any whose "tag" is already
// present, so rule sets / DNS servers stay unique across re-imports. An addition
// without a tag is de-duplicated by canonical JSON instead, so the helper keeps
// its idempotency contract even though current callers always tag their entries.
func mergeByTag(existing, additions []any) []any {
	seenTags := map[string]struct{}{}
	seenContent := map[string]struct{}{}
	for _, e := range existing {
		if m, ok := e.(map[string]any); ok {
			if tag, ok := m["tag"].(string); ok && tag != "" {
				seenTags[tag] = struct{}{}
			}
		}
		if key, err := json.Marshal(e); err == nil {
			seenContent[string(key)] = struct{}{}
		}
	}
	for _, a := range additions {
		tag := ""
		if m, ok := a.(map[string]any); ok {
			tag, _ = m["tag"].(string)
		}
		if tag != "" {
			if _, dup := seenTags[tag]; dup {
				continue
			}
			seenTags[tag] = struct{}{}
		} else if key, err := json.Marshal(a); err == nil {
			if _, dup := seenContent[string(key)]; dup {
				continue
			}
			seenContent[string(key)] = struct{}{}
		}
		existing = append(existing, a)
	}
	return existing
}

// resolveRoutingTarget maps an Xray outboundTag to an s-ui routing target. The
// targets map is built from the source outbounds (blackhole->reject action,
// freedom->direct outbound, wireguard/WARP->the endpoint tag). The fallback
// covers configs parsed without an outbounds list.
func resolveRoutingTarget(outboundTag string, targets map[string]string) (string, bool) {
	if t, ok := targets[outboundTag]; ok && t != "" {
		return t, true
	}
	switch strings.ToLower(outboundTag) {
	case "block", "blocked":
		return rejectTarget, true
	case "direct":
		return directOutboundTag, true
	}
	return "", false
}

func MapXrayRouting(raw string, targets map[string]string) (map[string]any, []string, int, int) {
	result := map[string]any{
		"route": map[string]any{
			"rules":    []any{},
			"rule_set": []any{},
		},
		"dns": map[string]any{},
	}
	if strings.TrimSpace(raw) == "" {
		return result, nil, 0, 0
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return result, []string{fmt.Sprintf("routing: invalid xrayConfig: %v", err)}, 0, 1
	}
	route := result["route"].(map[string]any)
	rulesOut := route["rules"].([]any)
	ruleSets := route["rule_set"].([]any)
	seenRuleSet := map[string]struct{}{}
	mapped := 0
	manual := 0
	var warnings []string
	routing, _ := cfg["routing"].(map[string]any)
	rules, _ := routing["rules"].([]any)
	for index, rawRule := range rules {
		rule, _ := rawRule.(map[string]any)
		if rule == nil {
			continue
		}
		if _, ok := rule["balancerTag"]; ok {
			manual++
			warnings = append(warnings, fmt.Sprintf("routing rule %d uses balancer; manual review required", index))
			continue
		}
		if _, ok := rule["attrs"]; ok {
			// Xray attrs match HTTP attributes/headers; sing-box has no
			// equivalent. Dropping them would silently broaden the match, so the
			// whole rule needs manual review.
			manual++
			warnings = append(warnings, fmt.Sprintf("routing rule %d uses attrs (HTTP attribute match) which sing-box does not support; manual review required", index))
			continue
		}
		outboundTag, _ := rule["outboundTag"].(string)
		outboundTag = strings.TrimSpace(outboundTag)
		if outboundTag == "" {
			manual++
			warnings = append(warnings, fmt.Sprintf("routing rule %d has no outboundTag; manual review required", index))
			continue
		}
		target, ok := resolveRoutingTarget(outboundTag, targets)
		if !ok {
			manual++
			warnings = append(warnings, fmt.Sprintf("routing rule %d outbound %q requires manual review", index, outboundTag))
			continue
		}
		next := map[string]any{}
		switch target {
		case dnsHijackTarget:
			// sing-box routes DNS via a rule action, not an outbound.
			next["action"] = "hijack-dns"
		case rejectTarget:
			// The migration never creates an outbound tagged "block" and sing-box
			// no longer auto-provides one, so emitting outbound:"block" would make
			// sing-box drop the matched connection at route time with
			// "outbound not found: block". The modern equivalent is a reject action.
			next["action"] = "reject"
		default:
			next["outbound"] = target
		}
		matched, matcherWarnings := applyRuleMatchers(index, rule, next, &ruleSets, seenRuleSet)
		warnings = append(warnings, matcherWarnings...)
		if !matched {
			manual++
			warnings = append(warnings, fmt.Sprintf("routing rule %d has no supported matchers; manual review required", index))
			continue
		}
		rulesOut = append(rulesOut, next)
		mapped++
	}
	if dns, ok := cfg["dns"].(map[string]any); ok {
		dnsOut, dnsWarnings := mapXrayDNS(dns, &ruleSets, seenRuleSet)
		warnings = append(warnings, dnsWarnings...)
		result["dns"] = dnsOut
	}
	route["rules"] = rulesOut
	route["rule_set"] = ruleSets
	return result, warnings, mapped, manual
}

func stringList(value any) []string {
	var result []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				result = append(result, s)
			}
		}
	case []string:
		result = append(result, v...)
	case string:
		if strings.TrimSpace(v) != "" {
			result = append(result, strings.TrimSpace(v))
		}
	}
	return result
}

func appendString(value any, item string) []string {
	if existing, ok := value.([]string); ok {
		return append(existing, item)
	}
	return []string{item}
}
