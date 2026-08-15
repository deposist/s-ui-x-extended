//go:build with_call

package capabilities

// Compiled only when -tags with_call is set; flips the detector entry true.
func init() { compiledBuildTags["with_call"] = true }
