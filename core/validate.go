package core

import (
	"context"

	sb "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
)

// ValidateConfig builds a sing-box instance from the supplied config without
// starting its lifecycle. It catches parse and construction failures while
// avoiding listener binds, outbound dials, and cache/ruleset downloads.
func ValidateConfig(sbConfig []byte) error {
	var opt option.Options
	ctx := context.Background()
	ctx = sb.Context(ctx, InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry())
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
