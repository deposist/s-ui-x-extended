package capabilities

import "testing"

func TestBuildTagsReportsEveryKnownTag(t *testing.T) {
	tags := BuildTags()
	if len(tags) != len(knownBuildTags) {
		t.Fatalf("BuildTags returned %d entries, want %d", len(tags), len(knownBuildTags))
	}
	for _, name := range knownBuildTags {
		if _, ok := tags[name]; !ok {
			t.Errorf("BuildTags missing known tag %q", name)
		}
	}
}

func TestTagCompiledEmptyIsAlwaysAvailable(t *testing.T) {
	if !tagCompiled("") {
		t.Error("an empty build tag (no tag required) must be available")
	}
}

// TestBuildAPIViewAvailabilityConsistent proves Available mirrors the build-tag
// status, that no-tag types are always available, and that alias rows are excluded.
func TestBuildAPIViewAvailabilityConsistent(t *testing.T) {
	view := BuildAPIView()
	tags := view.BuildTags
	seen := map[string]bool{}
	for _, in := range view.Inbounds {
		seen[in.Type] = true
		if in.BuildTag == "" {
			if !in.Available {
				t.Errorf("%s needs no build tag but is reported unavailable", in.Type)
			}
		} else if in.Available != tags[in.BuildTag] {
			t.Errorf("%s Available=%v but build tag %q=%v", in.Type, in.Available, in.BuildTag, tags[in.BuildTag])
		}
	}
	if seen["shadowsocks16"] {
		t.Error("alias type shadowsocks16 must not appear in the API view")
	}
	// Representative no-tag type is always available regardless of build.
	if !seen["vless"] {
		t.Error("expected vless in the API view")
	}
}
