package core

import (
	"context"

	sb "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

// registryContext returns a context carrying every registry sing-box needs to
// decode a config: without it the discriminated unions for inbounds,
// outbounds, endpoints, providers, DNS transports and services cannot resolve
// their concrete option types.
//
// This is the single definition shared by ValidateConfig and the
// rule-condition helpers, so a condition-only check decodes rules exactly the
// way a full validation does.
func registryContext(ctx context.Context) context.Context {
	return sb.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry())
}

// ValidateConfig builds a sing-box instance from the supplied config without
// starting its lifecycle. It catches parse and construction failures while
// avoiding listener binds, outbound dials, and cache/ruleset downloads.
func ValidateConfig(sbConfig []byte) error {
	var opt option.Options
	ctx := registryContext(context.Background())
	if err := opt.UnmarshalJSONContext(ctx, sbConfig); err != nil {
		return err
	}
	instance, err := NewBox(Options{
		Context: ctx,
		Options: opt,
	})
	if err != nil {
		return err
	}
	return instance.Close()
}
