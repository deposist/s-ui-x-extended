//go:build with_usbip

package capabilities

// Compiled only when -tags with_usbip is set; flips the detector entry true.
// Note: the usbip service also has a platform constraint (linux, darwin+cgo,
// windows) enforced by its registration stub; the tag detector only reports
// whether the tag was compiled in, not platform availability.
func init() { compiledBuildTags["with_usbip"] = true }
