//go:build with_sudoku

package capabilities

// Compiled only when -tags with_sudoku is set; flips the detector entry true.
func init() { compiledBuildTags["with_sudoku"] = true }
