//go:build with_oomkiller

package capabilities

// Compiled only when -tags with_oomkiller is set; flips the detector entry true.
func init() { compiledBuildTags["with_oomkiller"] = true }
