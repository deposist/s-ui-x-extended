package service

import (
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/ssrf"

	"gorm.io/gorm"
)

// validateCoreFailoverOutbound validates a native core failover outbound:
// outbounds must exist, strategy must be a recognised value, delay must be a
// valid duration if set.
func validateCoreFailoverOutbound(tx *gorm.DB, o model.Outbound) error {
	opts := optionsMapOf(o.Options)
	members, _ := opts["outbounds"].([]any)
	if len(members) == 0 {
		return common.NewError("core-failover needs at least one outbound")
	}
	for _, item := range members {
		tag, _ := item.(string)
		if tag == "" {
			return common.NewError("core-failover member must be a string tag")
		}
		if tag == o.Tag {
			return common.NewError("a core-failover outbound cannot reference itself")
		}
		var member model.Outbound
		if err := tx.Model(model.Outbound{}).Where("tag = ?", tag).First(&member).Error; err != nil {
			return common.NewErrorf("core-failover member %q does not exist", tag)
		}
		if isGroupType(member.Type) {
			return common.NewErrorf("core-failover member %q must be a plain outbound, not group type %q", tag, member.Type)
		}
	}
	strategy, _ := opts["strategy"].(string)
	switch strategy {
	case "", "sequential", "cycle":
	default:
		return common.NewErrorf("invalid core-failover strategy %q (must be sequential or cycle)", strategy)
	}
	delayStr, _ := opts["delay"].(string)
	if delayStr != "" {
		if _, err := time.ParseDuration(delayStr); err != nil {
			return common.NewErrorf("invalid core-failover delay %q: %v", delayStr, err)
		}
	}
	return nil
}

// validateFailoverGroup enforces the save-time rules for a Type:"failover" row:
// non-empty members, every member exists and is a plain outbound (not a group -
// which also rules out cycles), no self-reference, no duplicates, and a valid
// probe target / interval / hysteresis.
func validateFailoverGroup(tx *gorm.DB, o model.Outbound) error {
	opts, err := parseFailoverOptions(o.Options)
	if err != nil {
		return err
	}
	if len(opts.Outbounds) == 0 {
		return common.NewError("failover group needs at least one member")
	}

	var rows []model.Outbound
	if err := tx.Model(model.Outbound{}).Select("tag", "type").Find(&rows).Error; err != nil {
		return err
	}
	typeByTag := make(map[string]string, len(rows))
	for _, row := range rows {
		typeByTag[row.Tag] = row.Type
	}

	seen := make(map[string]struct{}, len(opts.Outbounds))
	for _, member := range opts.Outbounds {
		if member == o.Tag {
			return common.NewError("a failover group cannot reference itself")
		}
		if _, dup := seen[member]; dup {
			return common.NewErrorf("member %q is listed more than once", member)
		}
		seen[member] = struct{}{}
		memberType, exists := typeByTag[member]
		if !exists {
			return common.NewErrorf("failover member %q does not exist", member)
		}
		if memberType == "selector" || memberType == "urltest" || memberType == FailoverType {
			return common.NewErrorf("member %q is a group; failover members must be plain outbounds", member)
		}
	}

	if err := validateProbeTarget(opts.Failover.resolvedTarget()); err != nil {
		return err
	}
	if opts.Failover.Interval != "" {
		d, err := time.ParseDuration(opts.Failover.Interval)
		if err != nil {
			return common.NewErrorf("invalid probe interval %q", opts.Failover.Interval)
		}
		if d < MinInterval {
			return common.NewErrorf("probe interval must be >= %s", MinInterval)
		}
	}
	if opts.Failover.Hysteresis != 0 && opts.Failover.Hysteresis < 1 {
		return common.NewError("hysteresis must be >= 1")
	}
	if !validAllDownPolicy(opts.Failover.AllDownPolicy) {
		return common.NewErrorf("invalid all_down_policy %q (must be hold_current, block, or direct)", opts.Failover.AllDownPolicy)
	}
	return nil
}

// validateFallbackGroup enforces save-time rules for a Type:"fallback" row:
// non-empty members, every member exists, no self-reference, no duplicates.
// Fallback members must be plain outbounds (not groups) to prevent cycles.
func validateFallbackGroup(tx *gorm.DB, o model.Outbound) error {
	opts := optionsMapOf(o.Options)
	if opts == nil {
		return common.NewError("fallback group has no options")
	}
	members, _ := opts["outbounds"].([]any)
	if len(members) == 0 {
		return common.NewError("fallback group needs at least one member")
	}
	return validateGroupMembers(tx, o, members)
}

// validateSelectorURLTestGroup enforces save-time rules for selector and urltest
// groups: validates static members and provider references.
func validateSelectorURLTestGroup(tx *gorm.DB, o model.Outbound) error {
	opts := optionsMapOf(o.Options)
	if opts == nil {
		return nil
	}
	members, _ := opts["outbounds"].([]any)
	providers, _ := opts["providers"].([]any)
	useAllProviders, _ := opts["use_all_providers"].(bool)

	if len(members) == 0 && len(providers) == 0 && !useAllProviders {
		return common.NewError("group needs at least one member or provider")
	}

	if len(members) > 0 {
		if err := validateGroupMembers(tx, o, members); err != nil {
			return err
		}
	}

	if len(providers) > 0 {
		var providerRows []model.Provider
		if err := tx.Model(model.Provider{}).Select("tag").Find(&providerRows).Error; err != nil {
			return err
		}
		knownProviders := make(map[string]struct{}, len(providerRows))
		for _, p := range providerRows {
			knownProviders[p.Tag] = struct{}{}
		}
		seen := make(map[string]struct{}, len(providers))
		for _, item := range providers {
			tag, _ := item.(string)
			if tag == "" {
				continue
			}
			if _, dup := seen[tag]; dup {
				return common.NewErrorf("provider %q is listed more than once", tag)
			}
			seen[tag] = struct{}{}
			if _, ok := knownProviders[tag]; !ok {
				return common.NewErrorf("provider %q does not exist", tag)
			}
		}
	}
	return nil
}

// validateGroupMembers checks that all static group members exist, are not
// groups themselves (prevents cycles), are not self-references, and are not
// duplicated.
func validateGroupMembers(tx *gorm.DB, o model.Outbound, members []any) error {
	var rows []model.Outbound
	if err := tx.Model(model.Outbound{}).Select("tag", "type").Find(&rows).Error; err != nil {
		return err
	}
	typeByTag := make(map[string]string, len(rows))
	for _, row := range rows {
		typeByTag[row.Tag] = row.Type
	}
	seen := make(map[string]struct{}, len(members))
	for _, item := range members {
		member, _ := item.(string)
		if member == "" {
			continue
		}
		if member == o.Tag {
			return common.NewError("a group cannot reference itself")
		}
		if _, dup := seen[member]; dup {
			return common.NewErrorf("member %q is listed more than once", member)
		}
		seen[member] = struct{}{}
		memberType, exists := typeByTag[member]
		if !exists {
			return common.NewErrorf("member %q does not exist", member)
		}
		if isGroupType(memberType) {
			return common.NewErrorf("member %q is a group; group members must be plain outbounds", member)
		}
	}
	return nil
}

// validateProbeTarget accepts an absolute http(s) URL whose host may be a
// domain or IP. Unlike the panel's own outbound-check guard it does NOT reject
// private IPs - the probe is dialed THROUGH the member outbound, so a private
// target reachable via the tunnel is legitimate - but it still blocks
// infrastructure/cloud-metadata addresses (e.g. 169.254.169.254).
func validateProbeTarget(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return common.NewErrorf("invalid probe target: %v", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return common.NewError("probe target must be an http(s) URL")
	}
	if parsed.Hostname() == "" {
		return common.NewError("probe target must include a host")
	}
	if parsed.User != nil {
		return common.NewError("probe target must not include userinfo")
	}
	if addr, err := netip.ParseAddr(parsed.Hostname()); err == nil && ssrf.IsInfrastructureAddr(addr) {
		return common.NewError("probe target host is not allowed")
	}
	return nil
}
