package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeTrafficKeepsUnlimitedUseAndAvoidsDoubleCountingQuotas(t *testing.T) {
	db := initNodeTestDB(t)
	users, err := json.Marshal([]peerUser{{ClientID: 1, Name: "unlimited"}, {ClientID: 2, Name: "limited", Volume: 10},
		{ClientID: 1, Name: "unlimited"}, {ClientID: 1, Name: "probe", ProbeDestination: "127.0.0.1:443"}})
	if err != nil {
		t.Fatal(err)
	}
	panel := model.NodePanel{ID: 1, Role: "slave", UUID: "node", ManagedUsers: users}
	if err := db.Create(&panel).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	deltas := map[string]clientTrafficDelta{"unlimited": {up: 10, down: 30}, "limited": {up: 8}, "probe": {up: 7}}
	if err := addUnmeteredPeerTraffic(db, deltas); err != nil {
		t.Fatal(err)
	}
	report := localPeerTrafficReport(db, panel, now)
	if report == nil || len(report.Samples) != 2 || report.Samples[0] != (PeerTrafficSample{ClientID: 1, Used: 40}) || report.Samples[1].Used != 0 {
		t.Fatalf("unexpected local usage: %+v", report)
	}
	var counter model.PeerTraffic
	if err := db.Take(&counter, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || counter.Used != 0 {
		t.Fatalf("unlimited statistics debited the admission quota: %+v (%v)", counter, err)
	}
}

func TestNodeTrafficReportsOnlyUpdateConfirmedOwnAssignments(t *testing.T) {
	db := initNodeTestDB(t)
	now := time.Now().Truncate(time.Second)
	state := PeerSyncResponse{Users: []peerUser{{ClientID: 7, Volume: 1000}, {ClientID: 8, Volume: 1000}, {ClientID: 8, Volume: 1000}}}
	report := &PeerTrafficReport{SampledAt: now.Unix(), Samples: []PeerTrafficSample{{ClientID: 7, Used: 200}, {ClientID: 9, Used: 500}}}
	if err := acceptPeerTraffic(db, "node", state, report, now); err != nil {
		t.Fatal(err)
	}
	var rows []model.PeerTrafficSnapshot
	if err := db.Find(&rows).Error; err != nil || len(rows) != 1 || rows[0].ClientID != 7 || rows[0].Used != 200 {
		t.Fatalf("unexpected snapshots: %+v (%v)", rows, err)
	}
	report.Samples[0].Used = 0
	if err := acceptPeerTraffic(db, "node", state, report, now); err != nil {
		t.Fatal(err)
	}
	if err := db.Take(&rows[0]).Error; err != nil || rows[0].Used != 200 {
		t.Fatalf("replayed report changed usage: %+v (%v)", rows[0], err)
	}
}
