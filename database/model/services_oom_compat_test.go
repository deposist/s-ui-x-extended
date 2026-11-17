package model

import (
	"encoding/json"
	"testing"
)

func TestOOMKillerSaveDropsRetiredChecksOption(t *testing.T) {
	var service Service
	if err := service.UnmarshalJSON([]byte(`{"type":"oom-killer","tag":"oom","memory_limit":"512 MB","checks_before_limit":2}`)); err != nil {
		t.Fatal(err)
	}
	var persisted map[string]json.RawMessage
	if err := json.Unmarshal(service.Options, &persisted); err != nil {
		t.Fatal(err)
	}
	if _, exists := persisted["checks_before_limit"]; exists {
		t.Fatal("retired option still persisted after save")
	}
	if string(persisted["memory_limit"]) != `"512 MB"` {
		t.Fatal("supported OOM Killer option changed")
	}
}
