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
