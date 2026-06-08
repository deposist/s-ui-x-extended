//go:build with_sudoku

package capabilities

import "testing"

// TestBuildTagDetectedWhenCompiled proves the //go:build detector reports a tag as
// compiled when it actually is. Runs only in the tagged build (CI / Docker).
func TestBuildTagDetectedWhenCompiled(t *testing.T) {
	if !BuildTags()["with_sudoku"] {
		t.Fatal("with_sudoku is compiled in but BuildTags reports it false")
	}
	view := BuildAPIView()
	for _, in := range view.Inbounds {
		if in.Type == "sudoku" {
			if !in.Available {
				t.Fatal("sudoku must be Available when with_sudoku is compiled")
			}
			return
		}
	}
	t.Fatal("sudoku missing from API view")
}
