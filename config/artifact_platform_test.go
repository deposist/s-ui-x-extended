package config

import (
	"runtime"
	"testing"
)

func TestResolveArtifactPlatformRejectsNonLinuxBuilds(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Linux builds support Linux self-update artifacts")
	}
	previous := ArtifactPlatform
	previousGOOS := ArtifactGOOS
	ArtifactPlatform = runtime.GOARCH
	ArtifactGOOS = runtime.GOOS
	t.Cleanup(func() {
		ArtifactPlatform = previous
		ArtifactGOOS = previousGOOS
	})

	if got := ResolveArtifactPlatform(); got != "" {
		t.Fatalf("ResolveArtifactPlatform on %s = %q; want self-update disabled", runtime.GOOS, got)
	}
}
