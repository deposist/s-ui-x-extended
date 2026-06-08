//go:build with_tailscale

package capabilities

// Compiled only when -tags with_tailscale is set; flips the detector entry true.
func init() { compiledBuildTags["with_tailscale"] = true }
