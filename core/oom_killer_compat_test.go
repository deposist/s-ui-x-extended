//go:build with_oomkiller

package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/sagernet/sing-box/option"
)

func TestLegacyOOMKillerServiceOptionsRemainLoadable(t *testing.T) {
	service := model.Service{Type: "oom-killer", Tag: "oom", Options: json.RawMessage(`{"memory_limit":"512 MB","safety_margin":"64 MB","min_interval":"1s","max_interval":"5s","checks_before_limit":2}`)}
	payload, err := service.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry(), CertificateProviderRegistry())
	var options option.Options
	if err := options.UnmarshalJSONContext(ctx, []byte(`{"services":[`+string(payload)+`]}`)); err != nil {
		t.Fatalf("persisted OOM Killer service must load into the current core: %v", err)
	}
	if len(options.Services) != 1 {
		t.Fatalf("service was discarded instead of adapted: %d", len(options.Services))
	}
}
