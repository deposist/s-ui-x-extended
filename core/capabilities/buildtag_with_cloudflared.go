//go:build with_cloudflared

package capabilities

// Compiled only when -tags with_cloudflared is set; flips the detector entry true.
func init() { compiledBuildTags["with_cloudflared"] = true }
