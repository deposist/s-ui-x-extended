package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

type DoctorSeverity string

const (
	DoctorSeverityOK    DoctorSeverity = "ok"
	DoctorSeverityWarn  DoctorSeverity = "warn"
	DoctorSeverityError DoctorSeverity = "error"
)

type DoctorItem struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Severity DoctorSeverity `json:"severity"`
	Message  string         `json:"message"`
	Action   string         `json:"action,omitempty"`
	Details  any            `json:"details,omitempty"`
}

type DoctorReport struct {
	Status     DoctorSeverity `json:"status"`
	Summary    string         `json:"summary"`
	Items      []DoctorItem   `json:"items"`
	RanAt      int64          `json:"ranAt"`
	DurationMS int64          `json:"durationMs"`
}

type DoctorService struct {
	Runtime *Runtime
}

type DoctorClientRequest struct {
	ClientID uint   `json:"clientId"`
	Target   string `json:"target,omitempty"`
}

func (s *DoctorService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (s *DoctorService) Run(hostname string) DoctorReport {
	start := time.Now()
	var items []DoctorItem
	configService := NewConfigServiceWithRuntime(s.runtime())
	serverService := ServerService{Runtime: s.runtime()}

	rawConfig, err := configService.GetConfig("")
	if err != nil {
		items = append(items, doctorError("config-build", "Build sing-box config", "Unable to build config: "+err.Error(), "Fix database/config rows before restarting sing-box.", nil))
		return finishDoctorReport(start, items)
	}
	items = append(items, doctorOK("config-build", "Build sing-box config", "Full sing-box config was assembled from database rows.", nil))

	if err := core.ValidateConfig(*rawConfig); err != nil {
		items = append(items, doctorError("config-dry-check", "Dry config check", err.Error(), "Open the affected config section and fix the reported sing-box option.", nil))
	} else {
		items = append(items, doctorOK("config-dry-check", "Dry config check", "Config parses and constructs without starting sing-box.", nil))
	}

	if configService.IsCoreRunning() {
		items = append(items, doctorOK("core-running", "sing-box core", "Core is running.", nil))
	} else {
		items = append(items, doctorWarn("core-running", "sing-box core", "Core is not running.", "Start or restart sing-box after fixing config errors.", nil))
	}

	items = append(items, doctorReferenceChecks(*rawConfig)...)
	items = append(items, s.subscriptionChecks(hostname)...)
	items = append(items, s.recentLogCheck(serverService))
	items = append(items, s.outboundChecks(configService))

	return finishDoctorReport(start, items)
}

func (s *DoctorService) DiagnoseClient(req DoctorClientRequest, hostname string) (DoctorReport, error) {
	start := time.Now()
	if req.ClientID == 0 {
		return DoctorReport{}, common.NewError("clientId is required")
	}
	db := database.GetDB()
	if db == nil {
		return DoctorReport{}, common.NewError("database is not initialized")
	}

	var client model.Client
	if err := db.Model(model.Client{}).Where("id = ?", req.ClientID).First(&client).Error; err != nil {
		return DoctorReport{}, err
	}

	var items []DoctorItem
	now := time.Now().Unix()
	if client.Enable {
		items = append(items, doctorOK("client-enabled", "Client enabled", "Client is enabled.", nil))
	} else {
		items = append(items, doctorError("client-enabled", "Client enabled", "Client is disabled.", "Enable the client and save changes.", nil))
	}
	if client.Expiry == 0 || client.Expiry > now {
		items = append(items, doctorOK("client-expiry", "Expiry", "Client is not expired.", map[string]any{"expiry": client.Expiry}))
	} else {
		items = append(items, doctorError("client-expiry", "Expiry", "Client is expired.", "Extend the client expiry date.", map[string]any{"expiry": client.Expiry}))
	}
	used := client.Up + client.Down
	if client.Volume == 0 || used < client.Volume {
		items = append(items, doctorOK("client-traffic", "Traffic limit", "Client traffic is within the configured limit.", map[string]any{"used": used, "volume": client.Volume}))
	} else {
		items = append(items, doctorError("client-traffic", "Traffic limit", "Client reached the traffic limit.", "Increase volume or reset client traffic.", map[string]any{"used": used, "volume": client.Volume}))
	}

	inboundIDs, inboundItems := s.clientInboundChecks(client)
	items = append(items, inboundItems...)
	items = append(items, s.clientLinkChecks(client, inboundIDs)...)
	items = append(items, s.clientSubscriptionChecks(client)...)
	items = append(items, s.clientRuntimeChecks(client, req.Target)...)

	return finishDoctorReport(start, items), nil
}

func finishDoctorReport(start time.Time, items []DoctorItem) DoctorReport {
	status := DoctorSeverityOK
	errors := 0
	warnings := 0
	for _, item := range items {
		switch item.Severity {
		case DoctorSeverityError:
			errors++
			status = DoctorSeverityError
		case DoctorSeverityWarn:
			warnings++
			if status != DoctorSeverityError {
				status = DoctorSeverityWarn
			}
		}
	}
	summary := "All checks passed"
	if errors > 0 {
		summary = fmt.Sprintf("%d error(s), %d warning(s)", errors, warnings)
	} else if warnings > 0 {
		summary = fmt.Sprintf("%d warning(s)", warnings)
	}
	return DoctorReport{
		Status:     status,
		Summary:    summary,
		Items:      items,
		RanAt:      time.Now().Unix(),
		DurationMS: time.Since(start).Milliseconds(),
	}
}

func doctorOK(id, title, message string, details any) DoctorItem {
	return DoctorItem{ID: id, Title: title, Severity: DoctorSeverityOK, Message: message, Details: details}
}

func doctorWarn(id, title, message, action string, details any) DoctorItem {
	return DoctorItem{ID: id, Title: title, Severity: DoctorSeverityWarn, Message: message, Action: action, Details: details}
}

func doctorError(id, title, message, action string, details any) DoctorItem {
	return DoctorItem{ID: id, Title: title, Severity: DoctorSeverityError, Message: message, Action: action, Details: details}
}

func doctorReferenceChecks(rawConfig []byte) []DoctorItem {
	var cfg struct {
		DNS struct {
			Final   string           `json:"final"`
			Servers []map[string]any `json:"servers"`
			Rules   []map[string]any `json:"rules"`
		} `json:"dns"`
		Route struct {
			Final   string           `json:"final"`
			Rules   []map[string]any `json:"rules"`
			RuleSet []map[string]any `json:"rule_set"`
		} `json:"route"`
		Outbounds []map[string]any `json:"outbounds"`
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		return []DoctorItem{doctorError("reference-parse", "Reference scan", err.Error(), "Fix malformed config JSON.", nil)}
	}

	dnsTags := map[string]bool{}
	for i, server := range cfg.DNS.Servers {
		tag := stringField(server, "tag")
		if tag == "" {
			tag = strconv.Itoa(i)
		}
		dnsTags[tag] = true
	}
	outboundTags := map[string]bool{"direct": true, "block": true, "dns-out": true, "dns": true}
	for _, outbound := range cfg.Outbounds {
		if tag := stringField(outbound, "tag"); tag != "" {
			outboundTags[tag] = true
		}
	}
	for _, endpoint := range cfg.Endpoints {
		if tag := stringField(endpoint, "tag"); tag != "" {
			outboundTags[tag] = true
		}
	}

	var items []DoctorItem
	var missingDNS []string
	if cfg.DNS.Final != "" && !dnsTags[cfg.DNS.Final] {
		missingDNS = append(missingDNS, "dns.final -> "+cfg.DNS.Final)
	}
	for i, rule := range cfg.DNS.Rules {
		if server := stringField(rule, "server"); server != "" && !dnsTags[server] {
			missingDNS = append(missingDNS, fmt.Sprintf("dns.rules[%d].server -> %s", i, server))
		}
	}
	if len(missingDNS) > 0 {
		items = append(items, doctorError("dns-references", "DNS references", "DNS references missing server tags.", "Create the missing DNS server or update rules/final DNS.", missingDNS))
	} else {
		items = append(items, doctorOK("dns-references", "DNS references", "DNS final and rule server references resolve.", nil))
	}

	var missingOutbounds []string
	if cfg.Route.Final != "" && !outboundTags[cfg.Route.Final] {
		missingOutbounds = append(missingOutbounds, "route.final -> "+cfg.Route.Final)
	}
	for i, rule := range cfg.Route.Rules {
		missingOutbounds = appendMissingRouteOutbound(missingOutbounds, rule, i, outboundTags)
	}
	if len(missingOutbounds) > 0 {
		items = append(items, doctorError("route-references", "Route references", "Route references missing outbound/endpoint tags.", "Create the missing outbound or update the route rule/final outbound.", missingOutbounds))
	} else {
		items = append(items, doctorOK("route-references", "Route references", "Route final and rule outbound references resolve.", nil))
	}

	items = append(items, doctorRuleConditionChecks(cfg.Route.Rules, cfg.DNS.Rules)...)
	items = append(items, doctorRuleSetURLChecks(cfg.Route.RuleSet)...)
	items = append(items, doctorLocalRuleSetChecks(cfg.Route.RuleSet)...)
	return items
}

var ruleStructuralKeys = map[string]bool{"type": true, "mode": true, "rules": true, "invert": true}

// ruleHasConditions mirrors sing-box's rule validity check. Logical rules need
// non-empty valid branches; simple rules need one non-zero meaningful field.
func ruleHasConditions(rule map[string]any) bool {
	if stringField(rule, "type") == "logical" {
		nested, ok := rule["rules"].([]any)
		if !ok || len(nested) == 0 {
			return false
		}
		for _, item := range nested {
			sub, ok := item.(map[string]any)
			if !ok || !ruleHasConditions(sub) {
				return false
			}
		}
		return true
	}
	for key, value := range rule {
		if !ruleStructuralKeys[key] && ruleValueIsMeaningful(value) {
			return true
		}
	}
	return false
}

func ruleValueIsMeaningful(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case bool:
		return typed
	case float64:
		return typed != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func doctorRuleConditionChecks(routeRules, dnsRules []map[string]any) []DoctorItem {
	var empty []string
	scan := func(prefix string, rules []map[string]any) {
		for i, rule := range rules {
			if stringField(rule, "type") != "logical" {
				continue
			}
			nested, _ := rule["rules"].([]any)
			if len(nested) == 0 {
				empty = append(empty, fmt.Sprintf("%s[%d] has no sub-rules", prefix, i))
				continue
			}
			for j, item := range nested {
				sub, ok := item.(map[string]any)
				if !ok || !ruleHasConditions(sub) {
					empty = append(empty, fmt.Sprintf("%s[%d].rules[%d] has no conditions", prefix, i, j))
				}
			}
		}
	}
	scan("route.rules", routeRules)
	scan("dns.rules", dnsRules)
	if len(empty) > 0 {
		return []DoctorItem{doctorError("rule-conditions", "Rule conditions", "Some logical rules have no conditions, so sing-box will not start.", "Add a condition to each listed rule, or delete the rule.", empty)}
	}
	return []DoctorItem{doctorOK("rule-conditions", "Rule conditions", "Every logical rule has at least one condition.", nil)}
}

func appendMissingRouteOutbound(missing []string, rule map[string]any, index int, tags map[string]bool) []string {
	if outbound := stringField(rule, "outbound"); outbound != "" && !tags[outbound] {
		missing = append(missing, fmt.Sprintf("route.rules[%d].outbound -> %s", index, outbound))
	}
	if nested, ok := rule["rules"].([]any); ok {
		for _, item := range nested {
			if sub, ok := item.(map[string]any); ok {
				missing = appendMissingRouteOutbound(missing, sub, index, tags)
			}
		}
	}
	return missing
}

func doctorRuleSetURLChecks(ruleSets []map[string]any) []DoctorItem {
	var invalid []string
	for i, ruleSet := range ruleSets {
		if stringField(ruleSet, "type") != "remote" {
			continue
		}
		tag := stringField(ruleSet, "tag")
		rawURL := stringField(ruleSet, "url")
		if rawURL == "" {
			invalid = append(invalid, fmt.Sprintf("rule_set[%d] %q has empty url", i, tag))
			continue
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			invalid = append(invalid, fmt.Sprintf("rule_set[%d] %q has invalid url", i, tag))
			continue
		}
		if format := stringField(ruleSet, "format"); strings.HasSuffix(parsed.Path, ".srs") && format != "" && format != "binary" {
			invalid = append(invalid, fmt.Sprintf("rule_set[%d] %q uses .srs with non-binary format", i, tag))
		}
	}
	if len(invalid) > 0 {
		return []DoctorItem{doctorWarn("ruleset-urls", "Remote rule-set URLs", "Some remote rule-set URLs look unsafe or inconsistent.", "Use HTTPS raw URLs without credentials and binary format for .srs files.", invalid)}
	}
	return []DoctorItem{doctorOK("ruleset-urls", "Remote rule-set URLs", "Remote rule-set URLs have a safe shape.", nil)}
}

// doctorLocalRuleSetChecks verifies every local rule-set file is present and
// parseable.
//
// This is reported as an error rather than a warning because the core aborts
// startup on an unreadable local rule-set: the panel may look healthy while the
// next core restart would fail. Surfacing it here gives the operator a chance
// to re-download the assets before that happens.
func doctorLocalRuleSetChecks(ruleSets []map[string]any) []DoctorItem {
	var broken []string
	var found int
	for i, ruleSet := range ruleSets {
		if stringField(ruleSet, "type") != "local" {
			continue
		}
		found++
		tag := stringField(ruleSet, "tag")
		path := stringField(ruleSet, "path")
		if path == "" {
			broken = append(broken, fmt.Sprintf("rule_set[%d] %q has empty path", i, tag))
			continue
		}
		if err := VerifyRuleSetFile(path); err != nil {
			broken = append(broken, fmt.Sprintf("rule_set[%d] %q cannot be loaded from %s: %v", i, tag, path, err))
		}
	}
	if found == 0 {
		return nil
	}
	if len(broken) > 0 {
		return []DoctorItem{doctorError("ruleset-local-files", "Local rule-set files", "Some local rule-set files are missing or unreadable, which prevents sing-box from starting.", "Re-apply the regional preset to download the rule-set files again.", broken)}
	}
	return []DoctorItem{doctorOK("ruleset-local-files", "Local rule-set files", "All local rule-set files are present and loadable.", nil)}
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func (s *DoctorService) subscriptionChecks(hostname string) []DoctorItem {
	settingService := SettingService{}
	settings, err := settingService.GetAllSetting()
	if err != nil {
		return []DoctorItem{doctorError("subscription-settings", "Subscription settings", err.Error(), "Fix settings storage before serving subscriptions.", nil)}
	}
	var items []DoctorItem
	subURI, err := settingService.GetFinalSubURI(hostname)
	if err != nil || strings.TrimSpace(subURI) == "" {
		items = append(items, doctorWarn("subscription-uri", "Subscription URI", "Subscription URI cannot be resolved.", "Set subscription domain/URI in Settings.", nil))
	} else {
		items = append(items, doctorOK("subscription-uri", "Subscription URI", "Subscription URI resolves to "+subURI, nil))
	}
	enabledFormats := 0
	for _, key := range []string{"subLinkEnable", "subJsonEnable", "subClashEnable"} {
		if (*settings)[key] == "true" {
			enabledFormats++
		}
	}
	if enabledFormats == 0 {
		items = append(items, doctorError("subscription-formats", "Subscription formats", "All subscription formats are disabled.", "Enable at least one subscription format.", nil))
	} else {
		items = append(items, doctorOK("subscription-formats", "Subscription formats", fmt.Sprintf("%d subscription format(s) enabled.", enabledFormats), nil))
	}
	if (*settings)["subSecretRequired"] != "true" {
		items = append(items, doctorWarn("subscription-secret", "Subscription secret mode", "Legacy name-based subscription lookup is still allowed.", "Enable required subscription secrets to prevent name guessing.", nil))
	} else {
		items = append(items, doctorOK("subscription-secret", "Subscription secret mode", "Per-client subscription secrets are required.", nil))
	}
	return items
}

func (s *DoctorService) recentLogCheck(serverService ServerService) DoctorItem {
	logs, err := serverService.GetLogsFiltered("20", "warning", "core", "")
	if err != nil {
		return doctorWarn("core-logs", "Recent core logs", err.Error(), "Open Logs for details.", nil)
	}
	if len(logs) == 0 {
		return doctorOK("core-logs", "Recent core logs", "No recent core warnings/errors in the in-memory log buffer.", nil)
	}
	return doctorWarn("core-logs", "Recent core logs", "Recent core warnings/errors were found.", "Open Logs and inspect core warnings/errors.", firstNStrings(logs, 5))
}

func firstNStrings(values []string, n int) []string {
	if len(values) <= n {
		return values
	}
	return values[:n]
}

func (s *DoctorService) outboundChecks(configService *ConfigService) DoctorItem {
	return s.outboundChecksTarget(configService, "https://www.gstatic.com/generate_204")
}

func (s *DoctorService) outboundChecksTarget(configService *ConfigService, target string) DoctorItem {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "https://www.gstatic.com/generate_204"
	}
	outbounds, err := configService.OutboundService.GetAll()
	if err != nil {
		return doctorWarn("outbound-checks", "Outbound checks", err.Error(), "Open Outbounds and verify rows.", nil)
	}
	if !configService.IsCoreRunning() {
		return doctorWarn("outbound-checks", "Outbound checks", "Skipped because sing-box core is not running.", "Start sing-box before testing outbound latency.", nil)
	}
	tags := make([]string, 0, len(*outbounds))
	for _, outbound := range *outbounds {
		if tag, _ := outbound["tag"].(string); tag != "" {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	if len(tags) == 0 {
		return doctorWarn("outbound-checks", "Outbound checks", "No outbounds are configured.", "Add at least one proxy outbound for split routing.", nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	type result struct {
		Tag     string `json:"tag"`
		OK      bool   `json:"ok"`
		Error   string `json:"error,omitempty"`
		Delay   uint16 `json:"delay,omitempty"`
		Skipped bool   `json:"skipped,omitempty"`
	}
	results := make([]result, len(tags))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, tag := range tags {
		wg.Add(1)
		go func(index int, outboundTag string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = result{Tag: outboundTag, Error: ctx.Err().Error(), Skipped: true}
				return
			}
			check := configService.CheckOutboundWithContext(ctx, outboundTag, target)
			res := result{Tag: outboundTag, OK: check.OK, Error: check.Error, Delay: check.Delay}
			// A probe cancelled by the doctor's own time budget is "not tested",
			// not a genuine outbound failure - don't count it as failed.
			if !check.OK && ctx.Err() != nil {
				res.Skipped = true
			}
			results[index] = res
		}(i, tag)
	}
	wg.Wait()
	failed := 0
	skipped := 0
	for _, res := range results {
		switch {
		case res.Skipped:
			skipped++
		case !res.OK:
			failed++
		}
	}
	if failed > 0 {
		msg := fmt.Sprintf("%d outbound check(s) failed for %s.", failed, target)
		if skipped > 0 {
			msg += fmt.Sprintf(" %d not tested (time budget).", skipped)
		}
		return doctorWarn("outbound-checks", "Outbound checks", msg, "Open Outbounds and test failing tags individually.", results)
	}
	if skipped > 0 {
		return doctorWarn("outbound-checks", "Outbound checks", fmt.Sprintf("%d outbound(s) reached %s; %d not tested (time budget).", len(results)-skipped, target, skipped), "Re-run the doctor or test the remaining tags individually.", results)
	}
	return doctorOK("outbound-checks", "Outbound checks", fmt.Sprintf("%d outbound(s) reached %s.", len(results), target), results)
}

func (s *DoctorService) clientInboundChecks(client model.Client) ([]uint, []DoctorItem) {
	var inboundIDs []uint
	if err := json.Unmarshal(client.Inbounds, &inboundIDs); err != nil {
		return nil, []DoctorItem{doctorError("client-inbounds", "Client inbounds", "Client inbound list is malformed: "+err.Error(), "Re-save the client inbound list.", nil)}
	}
	if len(inboundIDs) == 0 {
		return inboundIDs, []DoctorItem{doctorError("client-inbounds", "Client inbounds", "Client has no inbounds.", "Assign at least one inbound to the client.", nil)}
	}
	var found []uint
	if err := database.GetDB().Model(model.Inbound{}).Where("id in ?", inboundIDs).Pluck("id", &found).Error; err != nil {
		return inboundIDs, []DoctorItem{doctorError("client-inbounds", "Client inbounds", err.Error(), "Open Clients and re-save inbound membership.", nil)}
	}
	if len(found) != len(inboundIDs) {
		return inboundIDs, []DoctorItem{doctorError("client-inbounds", "Client inbounds", "Some assigned inbounds no longer exist.", "Remove stale inbound ids or assign valid inbounds.", map[string]any{"assigned": inboundIDs, "found": found})}
	}
	return inboundIDs, []DoctorItem{doctorOK("client-inbounds", "Client inbounds", fmt.Sprintf("%d inbound(s) assigned.", len(inboundIDs)), inboundIDs)}
}

func (s *DoctorService) clientLinkChecks(client model.Client, inboundIDs []uint) []DoctorItem {
	var links []map[string]string
	if len(strings.TrimSpace(string(client.Links))) > 0 {
		if err := json.Unmarshal(client.Links, &links); err != nil {
			return []DoctorItem{doctorError("client-links", "Client links", "Client links are malformed: "+err.Error(), "Re-save the client to rebuild generated links.", nil)}
		}
	}
	if len(links) == 0 && len(inboundIDs) > 0 {
		return []DoctorItem{doctorWarn("client-links", "Client links", "No generated links are stored for this client.", "Re-save the client or its inbounds to regenerate links.", nil)}
	}
	return []DoctorItem{doctorOK("client-links", "Client links", fmt.Sprintf("%d link(s) stored for delivery.", len(links)), nil)}
}

func (s *DoctorService) clientSubscriptionChecks(client model.Client) []DoctorItem {
	settingService := SettingService{}
	var items []DoctorItem
	required, reqErr := settingService.GetSubSecretRequired()
	if client.SubSecret == "" {
		items = append(items, doctorWarn("client-sub-secret", "Subscription secret", "Client has no subscription secret yet.", "Rotate/re-save the client to generate a secret.", nil))
	} else {
		items = append(items, doctorOK("client-sub-secret", "Subscription secret", "Client has a subscription secret.", nil))
	}
	if reqErr != nil {
		items = append(items, doctorWarn("client-sub-secret-required", "Subscription lookup", "Could not read the subscription secret requirement: "+reqErr.Error(), "Check settings storage.", nil))
	} else if !required {
		items = append(items, doctorWarn("client-sub-secret-required", "Subscription lookup", "Legacy name lookup is allowed globally.", "Enable required subscription secrets in Settings.", nil))
	}
	linkOn, linkErr := settingService.GetSubLinkEnable()
	jsonOn, jsonErr := settingService.GetSubJsonEnable()
	clashOn, clashErr := settingService.GetSubClashEnable()
	if linkErr != nil || jsonErr != nil || clashErr != nil {
		// Do not assert "all formats disabled" when the settings read itself failed;
		// that would point the operator at the wrong fix.
		items = append(items, doctorWarn("client-sub-formats", "Subscription formats", "Could not read subscription format settings.", "Check settings storage before serving subscriptions.", nil))
		return items
	}
	enabled := map[string]bool{"link": linkOn, "json": jsonOn, "clash": clashOn}
	count := 0
	for _, ok := range enabled {
		if ok {
			count++
		}
	}
	if count == 0 {
		items = append(items, doctorError("client-sub-formats", "Subscription formats", "All subscription formats are disabled.", "Enable at least one subscription format.", enabled))
	} else {
		items = append(items, doctorOK("client-sub-formats", "Subscription formats", fmt.Sprintf("%d format(s) enabled.", count), enabled))
	}
	return items
}

func (s *DoctorService) clientRuntimeChecks(client model.Client, target string) []DoctorItem {
	configService := NewConfigServiceWithRuntime(s.runtime())
	var items []DoctorItem
	if configService.IsCoreRunning() {
		items = append(items, doctorOK("client-core", "sing-box core", "Core is running.", nil))
	} else {
		items = append(items, doctorWarn("client-core", "sing-box core", "Core is not running.", "Start sing-box before testing client traffic.", nil))
	}
	statsService := StatsService{Runtime: s.runtime()}
	onlines, err := statsService.GetOnlines()
	if err == nil {
		if doctorContainsString(onlines.User, client.Name) {
			items = append(items, doctorOK("client-online", "Online signal", "Client is currently online.", nil))
		} else {
			items = append(items, doctorWarn("client-online", "Online signal", "Client is not currently reported online.", "Ask the user to reconnect, then refresh online status.", map[string]any{"lastOnline": client.LastOnline, "lastIpCount": client.LastIPCount}))
		}
	}
	items = append(items, s.outboundChecksTarget(configService, target))
	return items
}

func doctorContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
