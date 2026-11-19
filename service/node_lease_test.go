package service

import (
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeLeaseTracksReceivedResponses(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	initNodeTestDB(t)
	nodes := &NodeService{peerNow: func() time.Time { return now }}
	if err := nodes.SetMaster(); err != nil {
		t.Fatal(err)
	}
	invite, err := nodes.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := nodes.RedeemInvite(invite, "lease test")
	if err != nil {
		t.Fatal(err)
	}
	input := PeerSyncRequest{LeaseCapable: true}
	sync := func() PeerSyncResponse {
		t.Helper()
		out, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, input)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	view := func(syncState, applyState string, until int64) {
		t.Helper()
		got, err := nodes.Get(peer.UUID)
		if err != nil || got.SyncState != syncState || got.ApplyState != applyState || got.AccessUntil != until {
			t.Fatalf("unexpected node state: %+v (%v)", got, err)
		}
	}
	view("waiting", "unknown", 0)
	first := sync()
	if first.Lease == nil || first.Lease.Until != now.Add(model.PeerAccessLease).Unix() {
		t.Fatal("Master did not issue a bounded lease")
	}
	input.AppliedRevision = first.Revision
	view("online", "pending", 0)
	forged := signPeerLease([]byte(nodeSecretDigest(peer.Token)), &model.Node{UUID: peer.UUID}, first.Revision, first.Lease.Until)
	input.LeaseReceipt = forged.Header()
	sync()
	view("online", "pending", 0)
	input.LeaseReceipt = first.Lease.Header()
	sync()
	view("online", "applied", first.Lease.Until)
	now = time.Unix(first.Lease.Until-1, 0)
	sync()
	view("online", "applied", first.Lease.Until)
	now = now.Add(time.Second)
	latest := sync()
	view("online", "pending", first.Lease.Until)
	input.LeaseReceipt = latest.Lease.Header()
	sync()
	view("online", "applied", latest.Lease.Until)
	input.LeaseCapable, input.LeaseReceipt = false, ""
	if out := sync(); out.Lease != nil {
		t.Fatal("legacy peer received an unknown JSON field")
	}
	view("online", "applied", latest.Lease.Until)
}
