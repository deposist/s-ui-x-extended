package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRejectUnknownCallOutboundDialFields(t *testing.T) {
	rejected := []json.RawMessage{
		json.RawMessage(`{"type":"call","tag":"call-1","server":"hello.watafafa.ru","server_port":443}`),
		json.RawMessage(`{"type":"call","tag":"call-1","server":"hello.watafafa.ru"}`),
		json.RawMessage(`{"type":"call","tag":"call-1","server_port":444}`),
	}
	for _, payload := range rejected {
		err := rejectUnknownCallOutboundDialFields(payload)
		if err == nil || !strings.Contains(err.Error(), "join_link") {
			t.Fatalf("call outbound with dial fields accepted: %s -> %v", payload, err)
		}
	}
	allowed := []json.RawMessage{
		json.RawMessage(`{"type":"call","tag":"call-1","join_link":"https://example.com/j"}`),
		json.RawMessage(`{"type":"vless","tag":"v-1","server":"hello.watafafa.ru","server_port":443}`),
		json.RawMessage(`{bad json`),
	}
	for _, payload := range allowed {
		if err := rejectUnknownCallOutboundDialFields(payload); err != nil {
			t.Fatalf("legitimate payload rejected: %s -> %v", payload, err)
		}
	}
}
