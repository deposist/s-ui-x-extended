package util

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// BenchmarkLinkGeneratorVless measures the link-generation hot path for a vless
// inbound with a ws transport across three addresses (the per-addr loop is the
// real work). Used to decide whether link generation warrants optimization
// before touching it (plan: profile before optimizing).
func BenchmarkLinkGeneratorVless(b *testing.B) {
	cfg := json.RawMessage(wellFormedClientConfig)
	in := &model.Inbound{
		Type:    "vless",
		Tag:     "node",
		Addrs:   json.RawMessage(`[{"server":"a.example.com","server_port":443,"remark":"-1"},{"server":"b.example.com","server_port":8443,"remark":"-2"},{"server":"c.example.com","server_port":2096,"remark":"-3"}]`),
		OutJson: json.RawMessage(`{}`),
		Options: json.RawMessage(`{"listen_port":443,"transport":{"type":"ws","path":"/ws","headers":{"Host":"a.example.com"}}}`),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = LinkGenerator(cfg, in, "example.com")
	}
}
