//go:build with_gvisor

package capabilities

// Compiled only when -tags with_gvisor is set; flips the detector entry true.
func init() { compiledBuildTags["with_gvisor"] = true }
