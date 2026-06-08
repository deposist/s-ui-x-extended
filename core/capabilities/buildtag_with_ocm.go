//go:build with_ocm

package capabilities

// Compiled only when -tags with_ocm is set; flips the detector entry true.
func init() { compiledBuildTags["with_ocm"] = true }
