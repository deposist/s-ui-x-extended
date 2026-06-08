//go:build with_mtproxy

package capabilities

// Compiled only when -tags with_mtproxy is set; flips the detector entry true.
func init() { compiledBuildTags["with_mtproxy"] = true }
