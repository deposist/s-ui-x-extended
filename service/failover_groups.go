package service

import (
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// Failover group shared constants - the single source of truth referenced by
// the assembler, validation, the failover manager and the frontend contract.
const (
	// FailoverType is the panel/DB outbound type for an auto-failover group. It
	// is transformed to a sing-box "selector" at the core-assembly seam.
	FailoverType = "failover"
	// DirectTag is the conventional tag of the seeded direct outbound used as
	// the all-down fallback.
	DirectTag = "direct"
	// DefaultProbeTarget is the out-of-the-box HTTP liveness-probe target.
	DefaultProbeTarget = "https://www.gstatic.com/generate_204"
	// DefaultInterval is the default per-group probe interval.
	DefaultInterval = 30 * time.Second
	// MinInterval is the lowest accepted probe interval.
	MinInterval = 5 * time.Second
	// DefaultHysteresis is the default consecutive-healthy-sample count required
	// before failing back to a higher-priority member.
	DefaultHysteresis = 2

	// All-down policy values. The default is conservative: hold the current
	// member rather than silently routing through direct.
	AllDownPolicyHoldCurrent = "hold_current"
	AllDownPolicyBlock       = "block"
	AllDownPolicyDirect      = "direct"
)

// validAllDownPolicy reports whether s is a recognised all-down policy.
func validAllDownPolicy(s string) bool {
	switch s {
	case AllDownPolicyHoldCurrent, AllDownPolicyBlock, AllDownPolicyDirect, "":
		return true
	}
	return false
}

// FailoverGroupConfig is the manager-facing view of a failover group: its
// ordered members plus the resolved probe settings.
type FailoverGroupConfig struct {
	Tag           string
	Members       []string
	ProbeTarget   string
	Interval      time.Duration
	Hysteresis    int
	Enabled       bool
	AllDownPolicy string
}

// LoadFailoverGroups returns every Type:"failover" group with its resolved probe
// settings. Malformed rows are skipped so one broken group never stalls the rest.
func LoadFailoverGroups(db *gorm.DB) ([]FailoverGroupConfig, error) {
	var rows []model.Outbound
	if err := db.Model(model.Outbound{}).Where("type = ?", FailoverType).Find(&rows).Error; err != nil {
		return nil, err
	}
	groups := make([]FailoverGroupConfig, 0, len(rows))
	for _, row := range rows {
		opts, err := parseFailoverOptions(row.Options)
		if err != nil || len(opts.Outbounds) == 0 {
			continue
		}
		groups = append(groups, FailoverGroupConfig{
			Tag:           row.Tag,
			Members:       opts.Outbounds,
			ProbeTarget:   opts.Failover.resolvedTarget(),
			Interval:      opts.Failover.resolvedInterval(),
			Hysteresis:    opts.Failover.resolvedHysteresis(),
			Enabled:       opts.Failover.ProbeEnabled(),
			AllDownPolicy: opts.Failover.resolvedAllDownPolicy(),
		})
	}
	return groups, nil
}

// DirectFallbackTag returns the tag of a direct-type outbound to use as the
// all-down fallback, preferring the conventional "direct" tag; "" when none.
func DirectFallbackTag(tx *gorm.DB) string {
	var tags []string
	if err := tx.Model(model.Outbound{}).Where("type = ?", "direct").Order("tag").Pluck("tag", &tags).Error; err != nil {
		return ""
	}
	for _, t := range tags {
		if t == DirectTag {
			return DirectTag
		}
	}
	if len(tags) > 0 {
		return tags[0]
	}
	return ""
}
