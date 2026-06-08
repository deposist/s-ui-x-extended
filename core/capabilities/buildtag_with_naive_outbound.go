//go:build with_naive_outbound

package capabilities

// Compiled only when -tags with_naive_outbound is set; flips the detector entry true.
func init() { compiledBuildTags["with_naive_outbound"] = true }
