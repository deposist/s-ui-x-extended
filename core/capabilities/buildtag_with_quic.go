//go:build with_quic

package capabilities

// Compiled only when -tags with_quic is set; flips the detector entry true.
func init() { compiledBuildTags["with_quic"] = true }
