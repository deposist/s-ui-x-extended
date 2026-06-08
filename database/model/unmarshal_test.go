package model

import (
	"encoding/json"
	"testing"
)

// TestOutboundEndpointUnmarshalTagNoPanic covers Q3: Outbound/Endpoint
// UnmarshalJSON must not panic when "tag" is absent or not a string (operator
// or import-supplied JSON), matching the safe comma-ok form used by Inbound.
func TestOutboundEndpointUnmarshalTagNoPanic(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"outbound missing tag", `{"type":"direct"}`},
		{"outbound non-string tag", `{"type":"direct","tag":123}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var o Outbound
			if err := json.Unmarshal([]byte(tc.data), &o); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o.Tag != "" {
				t.Fatalf("expected empty tag, got %q", o.Tag)
			}
		})
	}

	endpointCases := []struct {
		name string
		data string
	}{
		{"endpoint missing tag", `{"type":"wireguard"}`},
		{"endpoint non-string tag", `{"type":"wireguard","tag":true}`},
	}
	for _, tc := range endpointCases {
		t.Run(tc.name, func(t *testing.T) {
			var e Endpoint
			if err := json.Unmarshal([]byte(tc.data), &e); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.Tag != "" {
				t.Fatalf("expected empty tag, got %q", e.Tag)
			}
		})
	}
}
