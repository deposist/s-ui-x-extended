package core

import (
	"testing"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
)

// TestManagerCapabilityMatchesBuildTag keeps the reported capability honest: the
// panel decides whether to offer the Nodes section based on this flag, so a
// binary built with -tags with_manager must report it and a plain build must not.
func TestManagerCapabilityMatchesBuildTag(t *testing.T) {
	tags := capabilities.BuildTags()
	compiled, ok := tags["with_manager"]
	if !ok {
		t.Fatal("with_manager must always be reported, even when false")
	}
	if compiled != managerCompiled {
		t.Fatalf("with_manager capability = %v, want %v", compiled, managerCompiled)
	}
}
