//go:build with_manager

package capabilities

// Compiled only when -tags with_manager is set; flips the detector entry true.
func init() { compiledBuildTags["with_manager"] = true }
