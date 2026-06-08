//go:build with_acme

package capabilities

// Compiled only when -tags with_acme is set; flips the detector entry true.
func init() { compiledBuildTags["with_acme"] = true }
