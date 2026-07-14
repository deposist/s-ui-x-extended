package service

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

func TestClientIsActiveAtBoundaries(t *testing.T) {
	const now = int64(1_700_000_000)
	const maxInt64 = int64(1<<63 - 1)

	tests := []struct {
		name   string
		client model.Client
		want   bool
	}{
		{name: "unlimited", client: model.Client{Enable: true}, want: true},
		{name: "disabled", client: model.Client{}, want: false},
		{name: "expiry now", client: model.Client{Enable: true, Expiry: now}, want: false},
		{name: "expiry future", client: model.Client{Enable: true, Expiry: now + 1}, want: true},
		{name: "quota below", client: model.Client{Enable: true, Volume: 10, Up: 6, Down: 3}, want: true},
		{name: "quota equal", client: model.Client{Enable: true, Volume: 10, Up: 6, Down: 4}, want: false},
		{name: "quota sum overflow", client: model.Client{Enable: true, Volume: maxInt64 - 1, Up: maxInt64 - 5, Down: 10}, want: false},
		{name: "invalid negative counter", client: model.Client{Enable: true, Volume: 10, Up: -1}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientIsActiveAt(tt.client, now); got != tt.want {
				t.Fatalf("clientIsActiveAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
	// an empty slice, not be rejected - otherwise the link-regeneration paths
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

func TestDepleteClientsUsesActiveBoundarySemantics(t *testing.T) {
	initSettingTestDB(t)
	now := time.Now().Unix()
	clients := []model.Client{
		{
			Enable: true, Name: "expiry-now", Expiry: now,
			Inbounds: json.RawMessage(`[1]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{}`),
		},
		{
			Enable: true, Name: "quota-equal", Volume: 10, Up: 6, Down: 4,
			Inbounds: json.RawMessage(`[1]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{}`),
		},
		{
			Enable: true, Name: "still-active", Expiry: now + 3600, Volume: 10, Up: 6, Down: 3,
			Inbounds: json.RawMessage(`[1]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{}`),
		},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := (&ClientService{}).DepleteClients(); err != nil {
		t.Fatal(err)
	}

	var got []model.Client
	if err := database.GetDB().Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	state := make(map[string]bool, len(got))
	for _, client := range got {
		state[client.Name] = client.Enable
	}
	if state["expiry-now"] {
		t.Fatal("client expiring at the current Unix second should be depleted")
	}
	if state["quota-equal"] {
		t.Fatal("client at the exact traffic quota should be depleted")
	}
	if !state["still-active"] {
		t.Fatal("client below quota with a future expiry should stay active")
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
