//go:build with_utls

package capabilities

// Compiled only when -tags with_utls is set; flips the detector entry true.
func init() { compiledBuildTags["with_utls"] = true }
