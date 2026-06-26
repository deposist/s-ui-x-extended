package service

import (
	"sort"
	"sync"
	"time"
)

// OutboundHealthSnapshot is the bounded, operator-readable health view for one
// outbound or group member. It is kept in memory and never grows without bound.
type OutboundHealthSnapshot struct {
	Tag       string `json:"tag"`
	Status    string `json:"status"` // healthy | degraded | down | unknown
	DelayMs   uint16 `json:"delayMs,omitempty"`
	Error     string `json:"error,omitempty"`
	CheckedAt int64  `json:"checkedAt"`
}

// ProviderHealthSnapshot is the bounded health view for one provider.
type ProviderHealthSnapshot struct {
	Tag           string                   `json:"tag"`
	Status        string                   `json:"status"` // healthy | degraded | down | unknown
	UpdatedAt     int64                    `json:"updatedAt"`
	OutboundCount int                      `json:"outbounds"`
	HealthyCount  int                      `json:"healthyOutbounds"`
	LastError     string                   `json:"lastError,omitempty"`
	Members       []OutboundHealthSnapshot `json:"members,omitempty"`
}

// outboundHealthStore keeps a bounded number of recent outbound health
// snapshots. It is non-authoritative (failover_state is the crash-safe store).
var (
	outboundHealthMu         sync.RWMutex
	outboundHealthMap        = make(map[string]OutboundHealthSnapshot)
	outboundHealthMaxAge     = 10 * time.Minute
	outboundHealthMaxEntries = 1024

	providerHealthMu         sync.RWMutex
	providerHealthMap        = make(map[string]ProviderHealthSnapshot)
	providerHealthMaxAge     = 10 * time.Minute
	providerHealthMaxEntries = 512
	providerHealthMaxMembers = 32
)

// SetOutboundHealth records one outbound's latest health result.
func SetOutboundHealth(tag string, ok bool, delayMs uint16, errMsg string) {
	RecordOutboundHealth(tag, ok, delayMs, errMsg, time.Now())
}

// RecordOutboundHealth records one outbound's latest health result at a caller
// supplied timestamp and returns the stored snapshot.
func RecordOutboundHealth(tag string, ok bool, delayMs uint16, errMsg string, checkedAt time.Time) OutboundHealthSnapshot {
	status := "healthy"
	if !ok {
		status = "down"
	}
	snapshot := OutboundHealthSnapshot{
		Tag:       tag,
		Status:    status,
		DelayMs:   delayMs,
		Error:     truncateForHealth(errMsg, 200),
		CheckedAt: checkedAt.Unix(),
	}
	outboundHealthMu.Lock()
	outboundHealthMap[tag] = snapshot
	pruneOutboundHealthLocked(checkedAt.Unix())
	outboundHealthMu.Unlock()
	return snapshot
}

// OutboundHealthSnapshotFor returns the latest health snapshot for one outbound.
func OutboundHealthSnapshotFor(tag string) (OutboundHealthSnapshot, bool) {
	cutoff := time.Now().Add(-outboundHealthMaxAge).Unix()
	outboundHealthMu.Lock()
	defer outboundHealthMu.Unlock()
	s, ok := outboundHealthMap[tag]
	if !ok {
		return OutboundHealthSnapshot{}, false
	}
	if s.CheckedAt < cutoff {
		delete(outboundHealthMap, tag)
		return OutboundHealthSnapshot{}, false
	}
	return s, true
}

// AllOutboundHealthSnapshots returns a copy of all outbound health snapshots,
// pruned to entries checked within the retention window.
func AllOutboundHealthSnapshots() map[string]OutboundHealthSnapshot {
	nowUnix := time.Now().Unix()
	outboundHealthMu.Lock()
	defer outboundHealthMu.Unlock()
	pruneOutboundHealthLocked(nowUnix)
	out := make(map[string]OutboundHealthSnapshot, len(outboundHealthMap))
	for tag, s := range outboundHealthMap {
		out[tag] = s
	}
	return out
}

// PruneOutboundHealth removes snapshots for tags not in keep.
func PruneOutboundHealth(keep []string) {
	keepSet := make(map[string]struct{}, len(keep))
	for _, tag := range keep {
		keepSet[tag] = struct{}{}
	}
	outboundHealthMu.Lock()
	for tag := range outboundHealthMap {
		if _, ok := keepSet[tag]; !ok {
			delete(outboundHealthMap, tag)
		}
	}
	outboundHealthMu.Unlock()
}

// SetProviderHealth records one provider's aggregated health.
func SetProviderHealth(snapshot ProviderHealthSnapshot) {
	snapshot.LastError = truncateForHealth(snapshot.LastError, 200)
	if snapshot.UpdatedAt == 0 {
		snapshot.UpdatedAt = time.Now().Unix()
	}
	providerHealthMu.Lock()
	providerHealthMap[snapshot.Tag] = snapshot
	pruneProviderHealthLocked(snapshot.UpdatedAt)
	providerHealthMu.Unlock()
}

// AllProviderHealthSnapshots returns a copy of all provider health snapshots,
// pruned to entries updated within the retention window.
func AllProviderHealthSnapshots() map[string]ProviderHealthSnapshot {
	nowUnix := time.Now().Unix()
	providerHealthMu.Lock()
	defer providerHealthMu.Unlock()
	pruneProviderHealthLocked(nowUnix)
	out := make(map[string]ProviderHealthSnapshot, len(providerHealthMap))
	for tag, s := range providerHealthMap {
		out[tag] = s
	}
	return out
}

// PruneProviderHealth removes snapshots for tags not in keep.
func PruneProviderHealth(keep []string) {
	keepSet := make(map[string]struct{}, len(keep))
	for _, tag := range keep {
		keepSet[tag] = struct{}{}
	}
	providerHealthMu.Lock()
	for tag := range providerHealthMap {
		if _, ok := keepSet[tag]; !ok {
			delete(providerHealthMap, tag)
		}
	}
	providerHealthMu.Unlock()
}

// ResetHealthSnapshots clears all in-memory health snapshots. Called on DB reset.
func ResetHealthSnapshots() {
	outboundHealthMu.Lock()
	outboundHealthMap = make(map[string]OutboundHealthSnapshot)
	outboundHealthMu.Unlock()
	providerHealthMu.Lock()
	providerHealthMap = make(map[string]ProviderHealthSnapshot)
	providerHealthMu.Unlock()
}

func pruneOutboundHealthLocked(nowUnix int64) {
	cutoff := nowUnix - int64(outboundHealthMaxAge/time.Second)
	for tag, snapshot := range outboundHealthMap {
		if snapshot.CheckedAt < cutoff {
			delete(outboundHealthMap, tag)
		}
	}
	if len(outboundHealthMap) <= outboundHealthMaxEntries {
		return
	}
	entries := make([]healthEntryTime, 0, len(outboundHealthMap))
	for tag, snapshot := range outboundHealthMap {
		entries = append(entries, healthEntryTime{tag: tag, unix: snapshot.CheckedAt})
	}
	deleteOldestHealthEntries(outboundHealthMap, entries, len(outboundHealthMap)-outboundHealthMaxEntries)
}

func pruneProviderHealthLocked(nowUnix int64) {
	cutoff := nowUnix - int64(providerHealthMaxAge/time.Second)
	for tag, snapshot := range providerHealthMap {
		if snapshot.UpdatedAt < cutoff {
			delete(providerHealthMap, tag)
		}
	}
	if len(providerHealthMap) <= providerHealthMaxEntries {
		return
	}
	entries := make([]healthEntryTime, 0, len(providerHealthMap))
	for tag, snapshot := range providerHealthMap {
		entries = append(entries, healthEntryTime{tag: tag, unix: snapshot.UpdatedAt})
	}
	deleteOldestHealthEntries(providerHealthMap, entries, len(providerHealthMap)-providerHealthMaxEntries)
}

type healthEntryTime struct {
	tag  string
	unix int64
}

func deleteOldestHealthEntries[T any](m map[string]T, entries []healthEntryTime, count int) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].unix == entries[j].unix {
			return entries[i].tag < entries[j].tag
		}
		return entries[i].unix < entries[j].unix
	})
	for i := 0; i < count && i < len(entries); i++ {
		delete(m, entries[i].tag)
	}
}

func truncateForHealth(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 0 {
		return ""
	}
	if max <= 3 {
		return ""
	}
	limit := max - 3
	cut := 0
	for i := range s {
		if i > limit {
			break
		}
		cut = i
	}
	if cut == 0 {
		return "..."
	}
	return s[:cut] + "..."
}
