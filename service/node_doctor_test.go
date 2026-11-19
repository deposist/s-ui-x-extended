package service

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeDoctorShowsPerVPSQuotaAndStaleSamples(t *testing.T) {
	db := initNodeTestDB(t)
	client := nodeTestClient(t, db, "node doctor")
	client.Volume = 500
	now := time.Now().Truncate(time.Second)
	nodes := []model.Node{
		{UUID: "first", Name: "First", Enabled: true, EnrollmentDigest: "first-digest"},
		{UUID: "second", Name: "Second", Enabled: true, EnrollmentDigest: "second-digest"},
		{UUID: "third", Name: "Third", Enabled: true, EnrollmentDigest: "third-digest"},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	for i, node := range nodes {
		if err := db.Create(&model.PeerAssignment{NodeUUID: node.UUID, InboundID: 1, ClientID: client.Id}).Error; err != nil {
			t.Fatal(err)
		}
		at := now
		if i == 2 {
			at = now.Add(-peerTrafficFreshness - time.Second)
		}
		if err := db.Create(&model.PeerTrafficSnapshot{NodeUUID: node.UUID, ClientID: client.Id, Used: int64(i+1) * 250, SampledAt: at.Unix()}).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows, err := (&DoctorService{}).clientNodeDiagnostics(db, client, now)
	if err != nil || len(rows) != 3 {
		t.Fatalf("unexpected node diagnostics: %+v (%v)", rows, err)
	}
	if rows[0].Remaining == nil || *rows[0].Remaining != 250 || rows[1].Remaining == nil || *rows[1].Remaining != 0 ||
		rows[1].QuotaState != "exhausted" || rows[2].QuotaState != "stale" || rows[2].Remaining != nil || rows[2].SampledAt == 0 {
		t.Fatalf("unexpected per-node quotas: %+v", rows)
	}
}
