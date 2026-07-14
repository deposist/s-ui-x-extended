package capabilities

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var updateMatrix = flag.Bool("update", false, "rewrite docs/protocol-matrix.md from the manifest")

// TestProtocolMatrixDoc keeps docs/protocol-matrix.md in sync with the manifest.
// Regenerate after editing protocols.json:
//
//	go test ./core/capabilities -run TestProtocolMatrixDoc -update
func TestProtocolMatrixDoc(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "protocol-matrix.md")
	want := RenderMatrix()

	if *updateMatrix {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir docs: %v", err)
		}
		// #nosec G306 -- generated documentation is intentionally world-readable.
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatalf("write matrix: %v", err)
		}
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update): %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("docs/protocol-matrix.md is out of date with protocols.json.\nRegenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update")
	}
}

// TestRenderMatrixIncludesGroupTypes verifies that the rendered matrix
// includes all four group mode types with their assembledAs column.
func TestRenderMatrixIncludesGroupTypes(t *testing.T) {
	matrix := RenderMatrix()
	for _, groupType := range []string{"selector", "urltest", "fallback", "failover"} {
		if !strings.Contains(matrix, "| "+groupType+" |") {
			t.Fatalf("matrix does not include group type %q", groupType)
		}
	}
	// Panel-managed failover must show assembledAs=selector.
	if !strings.Contains(matrix, "| failover |") || !strings.Contains(matrix, "selector") {
		t.Fatal("matrix must show failover assembled as selector")
	}
	// The header must include the group column.
	if !strings.Contains(matrix, "| group |") {
		t.Fatal("matrix header must include a group column")
	}
}

// TestRenderMatrixIncludesProviderAndEndpointTypes verifies that the rendered
// matrix includes provider-relevant outbound types and all endpoint types.
func TestRenderMatrixIncludesProviderAndEndpointTypes(t *testing.T) {
	matrix := RenderMatrix()
	for _, endpointType := range []string{"wireguard", "tailscale", "vpn"} {
		if !strings.Contains(matrix, "| "+endpointType+" |") {
			t.Fatalf("matrix does not include endpoint type %q", endpointType)
		}
	}
	for _, serviceType := range []string{"resolved", "ssm-api", "derp", "ccm", "ocm", "oom-killer", "profiler"} {
		if !strings.Contains(matrix, "| "+serviceType+" |") {
			t.Fatalf("matrix does not include service type %q", serviceType)
		}
	}
}
