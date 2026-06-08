package capabilities

import (
	"flag"
	"os"
	"path/filepath"
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
