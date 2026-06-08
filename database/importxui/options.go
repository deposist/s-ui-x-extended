package importxui

import (
	"context"
	"fmt"
)

type Strategy string

const (
	StrategyMerge   Strategy = "merge"
	StrategyReplace Strategy = "replace"
	StrategySkip    Strategy = "skip"
)

type Options struct {
	Context        context.Context
	DryRun         bool
	Strategy       Strategy
	ReportPath     string
	SkipBackup     bool
	SkipAudit      bool
	Now            func() int64
	OnProgress     func(Progress)
	IncludeHistory bool
	IncludeRouting bool
	// Hostname is the address baked into each migrated client's subscription
	// links (mirrors the panel host used on a normal client save). Empty
	// leaves Links nil — the importer never emits a link with an empty host.
	Hostname string
}

func (o Options) normalized() (Options, error) {
	if o.Strategy == "" {
		o.Strategy = StrategyMerge
	}
	if err := o.Strategy.Validate(); err != nil {
		return o, err
	}
	return o, nil
}

type PlanOptions struct {
	Context         context.Context
	Strategy        Strategy
	IncludeSettings bool
	AdminMode       AdminMode
	OnlyNew         bool
	IncludeHistory  bool
	IncludeRouting  bool
}

func (o PlanOptions) normalized() (PlanOptions, error) {
	if o.Context == nil {
		o.Context = context.Background()
	}
	if o.Strategy == "" {
		o.Strategy = StrategyMerge
	}
	if err := o.Strategy.Validate(); err != nil {
		return o, err
	}
	if o.AdminMode == "" {
		o.AdminMode = AdminModeSkip
	}
	if err := o.AdminMode.Validate(); err != nil {
		return o, err
	}
	return o, nil
}

type ApplyOptions struct {
	Context    context.Context
	DryRun     bool
	SkipAudit  bool
	OnlyNew    bool
	Now        func() int64
	OnProgress func(Progress)
	// Hostname is the address baked into each migrated client's subscription
	// links. See Options.Hostname.
	Hostname string
}

func (o ApplyOptions) normalized() ApplyOptions {
	if o.Context == nil {
		o.Context = context.Background()
	}
	return o
}

type AdminMode string

const (
	AdminModeSkip          AdminMode = "skip"
	AdminModeNewPassword   AdminMode = "new_password"
	AdminModeResetRequired AdminMode = "reset_required"
)

func (m AdminMode) Validate() error {
	switch m {
	case AdminModeSkip, AdminModeNewPassword, AdminModeResetRequired:
		return nil
	default:
		return fmt.Errorf("invalid admin mode %q", m)
	}
}

func (s Strategy) Validate() error {
	switch s {
	case StrategyMerge, StrategyReplace, StrategySkip:
		return nil
	default:
		return fmt.Errorf("invalid strategy %q", s)
	}
}
