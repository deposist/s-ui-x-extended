//go:build with_grpc

package capabilities

// Compiled only when -tags with_grpc is set; flips the detector entry true.
func init() { compiledBuildTags["with_grpc"] = true }
