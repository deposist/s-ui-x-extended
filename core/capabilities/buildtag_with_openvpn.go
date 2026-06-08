//go:build with_openvpn

package capabilities

// Compiled only when -tags with_openvpn is set; flips the detector entry true.
func init() { compiledBuildTags["with_openvpn"] = true }
