package service

import (
	"errors"
	"testing"
	"time"
)

func TestNodeEnrollmentRedeemsOnlyOnce(t *testing.T) {
	db := initNodeTestDB(t)
	now := time.Now()
	code, err := createNodeInvite(db, now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := redeemNodeInvite(db, now, code, "first")
	if err != nil || first.UUID == "" || first.Token == "" || first.Token == code {
		t.Fatalf("redemption failed: %v", err)
	}
	if _, err := redeemNodeInvite(db, now, code, "duplicate"); !errors.Is(err, ErrInvalidNodeInvite) {
		t.Fatalf("duplicate redemption: %v", err)
	}
	code, err = createNodeInvite(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := redeemNodeInvite(db, now.Add(time.Hour), code, "late"); !errors.Is(err, ErrInvalidNodeInvite) {
		t.Fatalf("expired invitation: %v", err)
	}
	if _, err := enrolledNode(db, first.UUID, first.Token); err != nil {
		t.Fatalf("node credential rejected: %v", err)
	}
	if _, err := enrolledNode(db, first.UUID, code); !errors.Is(err, ErrInvalidNodeCredential) {
		t.Fatalf("unrelated credential: %v", err)
	}
}

func TestNodeMasterEndpoint(t *testing.T) {
	for _, address := range []string{"http://master.example", "https://user:pass@master.example", "https://master.example/?q=x", "https://master.example/#x"} {
		if _, err := nodeMasterEndpoint(address, "enroll"); err == nil {
			t.Errorf("accepted invalid URL %s", address)
		}
	}
	endpoint, err := nodeMasterEndpoint("https://master.example/panel/", "enroll")
	if err != nil || endpoint != "https://master.example/panel/node-peer/enroll" {
		t.Fatalf("unexpected endpoint %q: %v", endpoint, err)
	}
}

func TestSlaveRolePreservesLocalClients(t *testing.T) {
	db := initNodeTestDB(t)
	nodeTestClient(t, db, "keep-local")
	if err := checkSlaveEligibility(db); !errors.Is(err, ErrSlaveHasLocalData) {
		t.Fatalf("local client did not block role change: %v", err)
	}
	role, err := readNodePanel(db)
	if err != nil || role.Role != "standalone" {
		t.Fatalf("unexpected default role: %v, %v", role.Role, err)
	}
}
