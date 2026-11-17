package core

import (
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestBoxResetNetworkClosesTrackedConnections(t *testing.T) {
	cfg := []byte(`{
		"log":{"disabled":true},
		"dns":{"servers":[{"type":"local","tag":"local"}]},
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"}
	}`)
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry(), CertificateProviderRegistry())
	var opt option.Options
	if err := opt.UnmarshalJSONContext(ctx, cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	box, err := NewBox(Options{Context: ctx, Options: opt})
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	t.Cleanup(func() { _ = box.Close() })
	if err := box.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	client, server := net.Pipe()
	box.connection.TrackConn(client)
	t.Cleanup(func() {
		box.connection.CloseAll()
		_ = client.Close()
		_ = server.Close()
	})
	if count := box.connection.Count(); count != 1 {
		t.Fatalf("tracked connection count = %d, want 1", count)
	}

	box.network.ResetNetwork(context.Background())
	if count := box.connection.Count(); count != 0 {
		t.Fatalf("tracked connection count after ResetNetwork = %d, want 0", count)
	}
}
