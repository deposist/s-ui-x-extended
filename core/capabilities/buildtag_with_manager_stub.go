//go:build !with_manager

package capabilities

// Manager services are absent from this build. The stub exists so the tag is
// reported explicitly as false instead of being missing from the map.
func init() {}
