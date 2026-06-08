//go:build with_profiler

package capabilities

// Compiled only when -tags with_profiler is set; flips the detector entry true.
func init() { compiledBuildTags["with_profiler"] = true }
