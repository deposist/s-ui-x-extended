//go:build with_masque

package capabilities

// Compiled only when -tags with_masque is set; flips the detector entry true.
func init() { compiledBuildTags["with_masque"] = true }
