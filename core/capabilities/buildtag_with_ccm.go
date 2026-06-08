//go:build with_ccm

package capabilities

// Compiled only when -tags with_ccm is set; flips the detector entry true.
func init() { compiledBuildTags["with_ccm"] = true }
