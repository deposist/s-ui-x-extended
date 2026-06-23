package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// TestResetClientsClampsZeroResetDays pins the L6 fix: a client persisted (via
// apiv2/import, bypassing the UI guard) with auto_reset=true and reset_days=0 must
// not have NextReset collapse to dt and get its traffic wiped on every cron tick.
func TestResetClientsClampsZeroResetDays(t *testing.T) {
	initSettingTestDB(t)
	const now = int64(1_700_000_000)
	client := model.Client{
		Enable:    true,
		Name:      "zero-reset",
		Inbounds:  json.RawMessage(`[1]`),
		Links:     json.RawMessage(`[]`),
		Config:    json.RawMessage(`{}`),
		AutoReset: true,
		NextReset: now - 1,
		ResetDays: 0,
		Up:        10,
		Down:      20,
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := (&ClientService{}).ResetClients(database.GetDB(), now); err != nil {
		t.Fatal(err)
	}
	var got model.Client
	if err := database.GetDB().First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.NextReset != now+86400 {
		t.Fatalf("resetDays=0 must clamp to a 1-day period; got NextReset=%d want %d", got.NextReset, now+86400)
	}

	// Simulate traffic accruing, then a deplete run two minutes later: the client
	// must NOT re-match (next_reset is a day out) and its counters must survive.
	if err := database.GetDB().Model(&model.Client{}).Where("id = ?", client.Id).
		Updates(map[string]interface{}{"up": 5, "down": 7}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := (&ClientService{}).ResetClients(database.GetDB(), now+120); err != nil {
		t.Fatal(err)
	}
	var after model.Client
	if err := database.GetDB().First(&after, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if after.Up != 5 || after.Down != 7 {
		t.Fatalf("client was wrongly re-reset within a minute: up=%d down=%d", after.Up, after.Down)
	}
}
