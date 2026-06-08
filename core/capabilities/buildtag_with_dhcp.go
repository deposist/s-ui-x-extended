//go:build with_dhcp

package capabilities

// Compiled only when -tags with_dhcp is set; flips the detector entry true.
func init() { compiledBuildTags["with_dhcp"] = true }
