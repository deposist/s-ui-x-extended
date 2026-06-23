package service

import "sync"

// In-memory live status of every failover group, keyed by group tag. The
// failover manager writes it each due cycle (off the request path); stats.go
// reads it to merge into the onlines payload (the realtime + /api/load channel)
// so the UI needs no dedicated status poll. It is non-authoritative live data;
// the crash-safe history lives in the failover_state table.
var (
	failoverLiveMu sync.RWMutex
	failoverLive   = map[string]FailoverStatusEntry{}
)

// SetFailoverLiveStatus records one failover group's current live status.
func SetFailoverLiveStatus(entry FailoverStatusEntry) {
	failoverLiveMu.Lock()
	failoverLive[entry.Tag] = entry
	failoverLiveMu.Unlock()
}

// PruneFailoverLiveStatus drops live entries for groups not in keep, so the
// snapshot stays bounded as failover groups are deleted. keep=nil clears all.
func PruneFailoverLiveStatus(keep []string) {
	keepSet := make(map[string]struct{}, len(keep))
	for _, tag := range keep {
		keepSet[tag] = struct{}{}
	}
	failoverLiveMu.Lock()
	for tag := range failoverLive {
		if _, ok := keepSet[tag]; !ok {
			delete(failoverLive, tag)
		}
	}
	failoverLiveMu.Unlock()
}

// FailoverLiveSnapshot returns the per-group live status, sorted by tag for a
// deterministic payload.
func FailoverLiveSnapshot() map[string]FailoverStatusEntry {
	failoverLiveMu.RLock()
	out := make(map[string]FailoverStatusEntry, len(failoverLive))
	for tag, entry := range failoverLive {
		out[tag] = entry
	}
	failoverLiveMu.RUnlock()
	return out
}
