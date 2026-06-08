package core

import "strings"

const (
	TrackerValidatedSingBoxModule  = "github.com/sagernet/sing-box"
	TrackerValidatedSingBoxVersion = "v1.13.12"
	TrackerRevalidationPolicyDoc   = "docs/sing-box-tracker-revalidation.md"
)

var TrackerRevalidationChecks = []string{
	"RoutedConnection signature still matches sing-box adapter.RouterConnectionTracker",
	"RoutedPacketConnection signature still matches sing-box adapter.RouterConnectionTracker",
	"wrapped TCP connections always call Done exactly once on Close or terminal I/O error",
	"wrapped packet connections always call Done exactly once on Close or terminal I/O error",
	"Reset closes tracked connections and waits for active wrappers before replacing tracker state",
	"StatsTracker keeps counter pointers stable across Reset for already wrapped connections",
	"source IP extraction from adapter.InboundContext still uses metadata.Source.Addr",
}

type TrackerRevalidationStatus struct {
	Module           string
	ValidatedVersion string
	CurrentVersion   string
	Required         bool
	PolicyDoc        string
	Checks           []string
}

func SingBoxTrackerRevalidationStatus(currentVersion string) TrackerRevalidationStatus {
	currentVersion = normalizeTrackerVersion(currentVersion)
	validatedVersion := normalizeTrackerVersion(TrackerValidatedSingBoxVersion)
	return TrackerRevalidationStatus{
		Module:           TrackerValidatedSingBoxModule,
		ValidatedVersion: validatedVersion,
		CurrentVersion:   currentVersion,
		Required:         currentVersion == "" || currentVersion != validatedVersion,
		PolicyDoc:        TrackerRevalidationPolicyDoc,
		Checks:           append([]string(nil), TrackerRevalidationChecks...),
	}
}

func normalizeTrackerVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}
