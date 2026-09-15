package capabilities

import (
	"runtime"
	"testing"
)

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
		want := tagCompiled(in.BuildTag) && PlatformSupported(in.Platforms, runtime.GOOS)
		if in.Available != want {
			t.Errorf("%s Available=%v, want %v (build tag %q, platforms %v)", in.Type, in.Available, want, in.BuildTag, in.Platforms)
		}
		// A type whose build tag is absent must never be reported available.
		if in.BuildTag != "" && !tags[in.BuildTag] && in.Available {
			t.Errorf("%s is available although build tag %q is not compiled", in.Type, in.BuildTag)
		}
	}
	if seen["shadowsocks16"] {
		t.Error("alias type shadowsocks16 must not appear in the API view")
	}
	// Representative no-tag type is always available regardless of build.
	if !seen["vless"] {
		t.Error("expected vless in the API view")
	}

	// Endpoint and service editors gate their pickers on the same view, so every
	// manifest row of those categories must be reported with its availability -
	// a missing row would leave the picker offering a type this build cannot start.
	if len(view.Endpoints) != len(loaded.Endpoints) {
		t.Fatalf("API view reports %d endpoints, manifest has %d", len(view.Endpoints), len(loaded.Endpoints))
	}
	if len(view.Services) != len(loaded.Services) {
		t.Fatalf("API view reports %d services, manifest has %d", len(view.Services), len(loaded.Services))
	}
	for _, o := range view.Outbounds {
		want := tagCompiled(o.BuildTag) && PlatformSupported(o.Platforms, runtime.GOOS)
		if o.Available != want {
			t.Errorf("outbound %q Available=%v, want %v", o.Type, o.Available, want)
		}
	}
	for _, ep := range view.Endpoints {
		want := tagCompiled(ep.BuildTag) && PlatformSupported(ep.Platforms, runtime.GOOS)
		if ep.Available != want {
			t.Errorf("endpoint %q Available=%v, want %v (build tag %q, platforms %v)", ep.Type, ep.Available, want, ep.BuildTag, ep.Platforms)
		}
	}
	for _, svc := range view.Services {
		want := tagCompiled(svc.BuildTag) && PlatformSupported(svc.Platforms, runtime.GOOS)
		if svc.Available != want {
			t.Errorf("service %q Available=%v, want %v (build tag %q, platforms %v)", svc.Type, svc.Available, want, svc.BuildTag, svc.Platforms)
		}
	}
}

// TestPlatformRestrictedTypesFollowTheHost pins the platform half of the
// availability contract: REDIRECT/TPROXY compile on every OS but the core only
// implements them on their own platforms, so the panel must not offer them
// elsewhere. Without this the operator picks a type whose core construction
// fails with os.ErrInvalid at start.
func TestPlatformRestrictedTypesFollowTheHost(t *testing.T) {
	view := BuildAPIView()
	byType := map[string]APIInbound{}
	for _, in := range view.Inbounds {
		byType[in.Type] = in
	}
	for _, typ := range []string{"redirect", "tproxy"} {
		row, ok := byType[typ]
		if !ok {
			t.Fatalf("%s missing from the API view", typ)
		}
		if len(row.Platforms) == 0 {
			t.Fatalf("%s must declare its platform list", typ)
		}
		wantAvailable := PlatformSupported(row.Platforms, runtime.GOOS)
		if row.Available != wantAvailable {
			t.Errorf("%s Available=%v on %s, want %v (platforms %v)", typ, row.Available, runtime.GOOS, wantAvailable, row.Platforms)
		}
	}
	if runtime.GOOS != "linux" {
		if byType["tproxy"].Available {
			t.Errorf("tproxy must be unavailable on %s", runtime.GOOS)
		}
		if runtime.GOOS == "windows" && byType["redirect"].Available {
			t.Error("redirect must be unavailable on windows")
		}
	}
}
