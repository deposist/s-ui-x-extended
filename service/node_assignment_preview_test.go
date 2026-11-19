package service

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeAssignmentPreviewChecksPolicyAndRejectsStaleResult(t *testing.T) {
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	if err := nodes.SetMaster(); err != nil {
		t.Fatal(err)
	}
	invite, err := nodes.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := nodes.RedeemInvite(invite, "preview test")
	if err != nil {
		t.Fatal(err)
	}
	in := PeerSyncRequest{PublicHost: "slave.example", Inbounds: []PeerInbound{{ID: 15, Tag: "public", Type: "vless", Link: &peerPublicLink{Port: 443}}}}
	if _, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, in); err != nil {
		t.Fatal(err)
	}
	first := nodeTestClient(t, db, "first")
	incompatible := nodeTestClient(t, db, "incompatible")
	config := json.RawMessage(`{"vless":{"uuid":"c46a4d36-c8f2-4aa7-828a-8df136220995"}}`)
	if err := db.Model(&first).UpdateColumn("config", config).Error; err != nil {
		t.Fatal(err)
	}
	ids := []uint{first.Id, incompatible.Id, first.Id}
	preview, err := nodes.PreviewPeerClients(peer.UUID, 15, ids, true)
	if err != nil || preview.Changes != 1 || preview.Skipped != 1 || len(preview.Items) != 2 {
		t.Fatalf("unexpected preview: %+v (%v)", preview, err)
	}
	var count int64
	if err := db.Model(&model.PeerAssignment{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("preview changed assignments: %d (%v)", count, err)
	}
	if err := db.Model(&first).UpdateColumn("expiry", time.Now().Add(time.Hour).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.ApplyPeerClientPreview(peer.UUID, 15, ids, true, preview.Fingerprint); !errors.Is(err, ErrStalePeerPreview) {
		t.Fatalf("expected stale preview, got %v", err)
	}
	preview, err = nodes.PreviewPeerClients(peer.UUID, 15, ids, true)
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := nodes.ApplyPeerClientPreview(peer.UUID, 15, ids, true, preview.Fingerprint); err != nil || changed != 1 {
		t.Fatalf("changed %d assignments: %v", changed, err)
	}
	if changed, err := nodes.ApplyPeerClientPreview(peer.UUID, 15, ids, true, preview.Fingerprint); err != nil || changed != 0 {
		t.Fatalf("retry changed %d assignments: %v", changed, err)
	}
}
