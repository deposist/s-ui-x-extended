package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPanelFailoverTerminography ensures the docs plan does not describe
// panel-managed failover as generic session recovery. The file is in .gitignore
// (local planning artifact), so the test skips when the file is absent.
func TestPanelFailoverTerminography(t *testing.T) {
	path := filepath.Join("..", "docs", "plans", "panel-control-plane-without-core-changes-plan.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("docs plan not present (likely .gitignore'd): %v", err)
	}
	text := string(data)

	// The plan must explicitly state that generic failover does not preserve
	// existing sessions.
	if !strings.Contains(text, "Existing TCP or UDP sessions may break") {
		t.Error("docs plan must state that existing sessions may break")
	}

	// The plan must explicitly warn against promising session recovery for
	// arbitrary outbounds.
	if !strings.Contains(text, "Do not promise session recovery for arbitrary outbounds") {
		t.Error("docs plan must warn against promising session recovery")
	}

	// The plan must distinguish core failover protocol from panel-managed
	// failover.
	if !strings.Contains(text, "panel-managed failover") {
		t.Error("docs plan must use the term 'panel-managed failover'")
	}

	// The assembled-as wording must be present so the core boundary is clear.
	if !strings.Contains(text, "assembled as a normal `selector`") && !strings.Contains(text, "assembled as a core `selector`") && !strings.Contains(text, "assembled as `selector`") {
		t.Error("docs plan must mention that panel failover is assembled as a core selector")
	}
}
