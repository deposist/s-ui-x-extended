package importxui

import (
	"encoding/json"
	"testing"
)

func TestMapTransport_HTTPUpgradeAndH2(t *testing.T) {
	httpUpgrade, warnings := mapTransport("inbound", "in", xuiStreamSettings{
		Network: "httpupgrade",
		HTTPUPSettings: map[string]any{
			"host": "example.com",
			"path": "/up",
		},
	})
	if len(warnings) != 0 || httpUpgrade["type"] != "httpupgrade" || httpUpgrade["host"] != "example.com" {
		t.Fatalf("unexpected httpupgrade mapping: %#v warnings=%v", httpUpgrade, warnings)
	}
	h2, warnings := mapTransport("inbound", "in", xuiStreamSettings{
		Network: "h2",
		HTTPSettings: map[string]any{
			"host": []any{"a.example"},
			"path": "/h2",
		},
	})
	if len(warnings) != 0 || h2["type"] != "http" {
		t.Fatalf("unexpected h2 mapping: %#v warnings=%v", h2, warnings)
	}
}

func TestMapTransport_SplitHTTPWarnsAndKCPQUICSkip(t *testing.T) {
	split, warnings := mapTransport("inbound", "in", xuiStreamSettings{Network: "splithttp"})
	if split["type"] != "httpupgrade" || len(warnings) == 0 {
		t.Fatalf("splithttp should map with warning, got %#v warnings=%v", split, warnings)
	}
	for _, network := range []string{"kcp", "quic"} {
		row := xuiInboundRow{
			ID:             1,
			Protocol:       "vless",
			Port:           443,
			Tag:            "bad-" + network,
			Settings:       json.RawMessage(`{"clients":[{"email":"a","id":"00000000-0000-4000-8000-000000000000"}]}`),
			StreamSettings: json.RawMessage(`{"network":"` + network + `","security":"none"}`),
		}
		mapped, err := mapInbound(row, 0, nil, "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		if mapped.Inbound.Type != "" || len(mapped.Warnings) == 0 {
			t.Fatalf("%s should be skipped with warning: %#v", network, mapped)
		}
	}
}
