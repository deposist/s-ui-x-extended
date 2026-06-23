package service

import "testing"

// The in-memory live snapshot the failover manager populates must surface through
// the onlines payload (the realtime + /api/load channel) so the UI can drop its
// dedicated status poll.
func TestGetOnlinesIncludesFailoverSnapshot(t *testing.T) {
	t.Cleanup(func() { PruneFailoverLiveStatus(nil) })

	SetFailoverLiveStatus(FailoverStatusEntry{
		Tag:     "g",
		Active:  "b",
		AllDown: false,
		Members: []FailoverMemberStatus{
			{Tag: "a", Healthy: false, Priority: 0},
			{Tag: "b", Healthy: true, Priority: 1},
		},
	})

	on, err := (&StatsService{}).GetOnlines()
	if err != nil {
		t.Fatal(err)
	}
	if len(on.Failover) != 1 {
		t.Fatalf("onlines.failover len = %d, want 1", len(on.Failover))
	}
	entry := on.Failover["g"]
	if entry.Tag != "g" || entry.Active != "b" || entry.AllDown {
		t.Fatalf("onlines.failover[g] = %#v, want g/active=b/!allDown", entry)
	}
	if len(entry.Members) != 2 || !entry.Members[1].Healthy || entry.Members[0].Healthy {
		t.Fatalf("onlines.failover member health not surfaced: %#v", entry.Members)
	}
}

// Pruning keeps the snapshot bounded to groups that still exist.
func TestPruneFailoverLiveStatusDropsRemovedGroups(t *testing.T) {
	t.Cleanup(func() { PruneFailoverLiveStatus(nil) })

	SetFailoverLiveStatus(FailoverStatusEntry{Tag: "keep"})
	SetFailoverLiveStatus(FailoverStatusEntry{Tag: "gone"})
	PruneFailoverLiveStatus([]string{"keep"})

	snap := FailoverLiveSnapshot()
	entry, ok := snap["keep"]
	if len(snap) != 1 || !ok || entry.Tag != "keep" {
		t.Fatalf("snapshot after prune = %#v, want only keep", snap)
	}
}
