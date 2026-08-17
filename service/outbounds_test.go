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
		err := rejectInvalidCallOutbound(payload)
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
		if err := rejectInvalidCallOutbound(payload); err != nil {
			t.Fatalf("legitimate payload rejected: %s -> %v", payload, err)
		}
	}
}

func TestRejectInvalidCallOutboundRequiresJoinLink(t *testing.T) {
	rejected := []json.RawMessage{
		json.RawMessage(`{"type":"call","tag":"call-1"}`),
		json.RawMessage(`{"type":"call","tag":"call-1","join_link":"   "}`),
	}
	for _, payload := range rejected {
		err := rejectInvalidCallOutbound(payload)
		if err == nil || !strings.Contains(err.Error(), "join_link") {
			t.Fatalf("call outbound without join_link accepted: %s -> %v", payload, err)
		}
	}
	if err := rejectInvalidCallOutbound(json.RawMessage(`{"type":"call","tag":"call-1","join_link":"https://example.com/j"}`)); err != nil {
		t.Fatalf("call outbound with join_link rejected: %v", err)
	}
}
