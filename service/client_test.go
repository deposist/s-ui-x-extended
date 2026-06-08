package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/gorm"
)

func TestDecodeClientInbounds(t *testing.T) {
	got, ok := decodeClientInbounds(7, []byte(`[1,2,3]`), "test")
	if !ok {
		t.Fatal("valid inbounds should decode")
	}
	if want := []uint{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected inbounds: %#v, want %#v", got, want)
	}

	if _, ok := decodeClientInbounds(7, []byte(`{`), "test"); ok {
		t.Fatal("invalid inbounds should be rejected")
	}
}

func TestDecodeClientLinks(t *testing.T) {
	got, ok := decodeClientLinks(7, []byte(`[{"remark":"in","type":"local","uri":"vless://example"}]`), "test")
	if !ok {
		t.Fatal("valid links should decode")
	}
	want := []map[string]string{{
		"remark": "in",
		"type":   "local",
		"uri":    "vless://example",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected links: %#v, want %#v", got, want)
	}

	if _, ok := decodeClientLinks(7, []byte(`{`), "test"); ok {
		t.Fatal("invalid links should be rejected")
	}

	// A NULL/empty Links column (e.g. a freshly migrated client) must decode to
	// an empty slice, not be rejected — otherwise the link-regeneration paths
	// skip the client and its inbounds never reach the subscription.
	for _, raw := range []json.RawMessage{nil, []byte(""), []byte("  "), []byte("null")} {
		got, ok := decodeClientLinks(7, raw, "test")
		if !ok {
			t.Fatalf("empty links %q should decode, not be rejected", raw)
		}
		if len(got) != 0 {
			t.Fatalf("empty links %q decoded to %#v, want empty", raw, got)
		}
	}
}

func TestDepleteClientsTrafficLimitAvoidsInt64Overflow(t *testing.T) {
	initSettingTestDB(t)
	const maxInt64 = int64(1<<63 - 1)
	clients := []model.Client{
		{
			Enable:   true,
			Name:     "overflow-over-limit",
			Inbounds: json.RawMessage(`[1]`),
			Links:    json.RawMessage(`[]`),
			Config:   json.RawMessage(`{}`),
			Up:       maxInt64 - 5,
			Down:     10,
			Volume:   maxInt64 - 1,
		},
		{
			Enable:   true,
			Name:     "near-limit",
			Inbounds: json.RawMessage(`[1]`),
			Links:    json.RawMessage(`[]`),
			Config:   json.RawMessage(`{}`),
			Up:       maxInt64 - 10,
			Down:     5,
			Volume:   maxInt64 - 1,
		},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := (&ClientService{}).DepleteClients(); err != nil {
		t.Fatal(err)
	}

	var got []model.Client
	if err := database.GetDB().Order("name").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	state := map[string]bool{}
	for _, client := range got {
		state[client.Name] = client.Enable
	}
	if state["overflow-over-limit"] {
		t.Fatal("overflowing total traffic should be depleted")
	}
	if !state["near-limit"] {
		t.Fatal("client below volume should stay enabled")
	}
}

func TestResetClientsUsesColumnUpdatesAndPreservesIndependentFields(t *testing.T) {
	initSettingTestDB(t)
	const now = int64(1_700_000_000)
	client := model.Client{
		Enable:    true,
		Name:      "reset-me",
		Inbounds:  json.RawMessage(`[1]`),
		Links:     json.RawMessage(`[]`),
		Config:    json.RawMessage(`{}`),
		AutoReset: true,
		NextReset: now - 1,
		ResetDays: 1,
		Up:        10,
		Down:      20,
		TotalUp:   100,
		TotalDown: 200,
		Volume:    300,
		Expiry:    400,
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	const callbackName = "test:manual-client-update-before-reset"
	triggered := false
	if err := database.GetDB().Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if triggered || tx.Statement.Table != "clients" {
			return
		}
		triggered = true
		if err := tx.Session(&gorm.Session{NewDB: true}).Model(model.Client{}).Where("id = ?", client.Id).Updates(map[string]interface{}{
			"volume": int64(777),
			"expiry": int64(888),
		}).Error; err != nil {
			t.Errorf("manual update failed: %v", err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = database.GetDB().Callback().Update().Remove(callbackName)
	})

	if _, err := (&ClientService{}).ResetClients(database.GetDB(), now); err != nil {
		t.Fatal(err)
	}

	var got model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Volume != 777 || got.Expiry != 888 {
		t.Fatalf("independent fields were overwritten: volume=%d expiry=%d", got.Volume, got.Expiry)
	}
	if got.Up != 0 || got.Down != 0 || got.TotalUp != 110 || got.TotalDown != 220 || got.NextReset != now+86400 {
		t.Fatalf("reset fields not updated correctly: %#v", got)
	}
}
