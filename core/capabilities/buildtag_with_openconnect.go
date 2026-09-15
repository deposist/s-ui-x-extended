//go:build with_openconnect

package capabilities

// Compiled only when -tags with_openconnect is set; flips the detector entry true.
func init() { compiledBuildTags["with_openconnect"] = true }
