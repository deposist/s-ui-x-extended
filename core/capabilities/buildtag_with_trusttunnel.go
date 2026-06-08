//go:build with_trusttunnel

package capabilities

// Compiled only when -tags with_trusttunnel is set; flips the detector entry true.
func init() { compiledBuildTags["with_trusttunnel"] = true }
