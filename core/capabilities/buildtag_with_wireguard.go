//go:build with_wireguard

package capabilities

// Compiled only when -tags with_wireguard is set; flips the detector entry true.
func init() { compiledBuildTags["with_wireguard"] = true }
