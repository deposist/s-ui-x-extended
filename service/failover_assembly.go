package service

import (
	"encoding/json"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// failoverProbe is the per-group liveness-probe config stored under the
// "failover" key of a Type:"failover" outbound's Options blob. It drives only
// the failover manager and never reaches the core (stripped at assembly).
type failoverProbe struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	ProbeTarget string `json:"probe_target,omitempty"`
	Interval    string `json:"interval,omitempty"`
	Hysteresis  int    `json:"hysteresis,omitempty"`
}

// failoverOptions is the Options blob of a Type:"failover" outbound. Outbounds
// is the ordered priority list (index 0 = primary).
type failoverOptions struct {
	Outbounds                 []string      `json:"outbounds"`
	Default                   string        `json:"default,omitempty"`
	InterruptExistConnections *bool         `json:"interrupt_exist_connections,omitempty"`
	Failover                  failoverProbe `json:"failover"`
}

func parseFailoverOptions(options json.RawMessage) (failoverOptions, error) {
	var opts failoverOptions
	if len(options) == 0 {
		return opts, common.NewError("failover group has no options")
	}
	if err := json.Unmarshal(options, &opts); err != nil {
		return opts, err
	}
	return opts, nil
}

// ProbeEnabled reports whether the manager should probe and switch this group.
func (p failoverProbe) ProbeEnabled() bool {
	return p.Enabled == nil || *p.Enabled
}

// resolvedInterval returns the configured probe interval, defaulting/clamping.
func (p failoverProbe) resolvedInterval() time.Duration {
	if p.Interval == "" {
		return DefaultInterval
	}
	d, err := time.ParseDuration(p.Interval)
	if err != nil || d < MinInterval {
		return DefaultInterval
	}
	return d
}

func (p failoverProbe) resolvedHysteresis() int {
	if p.Hysteresis < 1 {
		return DefaultHysteresis
	}
	return p.Hysteresis
}

func (p failoverProbe) resolvedTarget() string {
	if p.ProbeTarget == "" {
		return DefaultProbeTarget
	}
	return p.ProbeTarget
}

// assembleFailoverForCore turns a persisted Type:"failover" row into the clean
// sing-box "selector" JSON the core accepts: the failover metadata is stripped
// (sing-box rejects unknown option keys via badjson DisallowUnknownFields), the
// direct fallback member is appended when available, and default is pinned to
// the primary so a cold start before the manager's first tick is deterministic.
func assembleFailoverForCore(o model.Outbound, directTag string) (json.RawMessage, error) {
	opts, err := parseFailoverOptions(o.Options)
	if err != nil {
		return nil, err
	}
	if len(opts.Outbounds) == 0 {
		return nil, common.NewErrorf("failover group %q has no members", o.Tag)
	}
	members := append([]string(nil), opts.Outbounds...)
	if directTag != "" && !stringSliceContains(members, directTag) {
		members = append(members, directTag)
	}
	selector := map[string]any{
		"type":      "selector",
		"tag":       o.Tag,
		"outbounds": members,
		"default":   opts.Outbounds[0],
	}
	if opts.InterruptExistConnections != nil {
		selector["interrupt_exist_connections"] = *opts.InterruptExistConnections
	}
	return json.Marshal(selector)
}

func stringSliceContains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
