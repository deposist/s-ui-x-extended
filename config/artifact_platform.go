package config

import "runtime"

// ArtifactPlatform is the release artifact platform suffix this binary was built
// for (e.g. "amd64", "arm64", "armv7", "386", "s390x"), matching the
// `s-ui-linux-<platform>.tar.gz` asset naming in .github/workflows/release.yml.
//
// It is injected at build time via:
//
//	-ldflags "-X github.com/deposist/s-ui-x-extended/config.ArtifactPlatform=<platform>"
//
// This is the only reliable way to disambiguate the ARM variants (armv5/armv6/
// armv7 all report runtime.GOARCH == "arm"): the chosen GOARM is a build-time
// decision the runtime does not expose. For non-ARM targets ResolveArtifactPlatform
// can derive the suffix from runtime.GOARCH alone, so the injection is optional
// there but recommended for consistency.
var ArtifactPlatform string

// ArtifactGOOS is injected alongside ArtifactPlatform for release builds. It is
// overridable in tests so non-Linux hosts can exercise Linux release metadata.
var ArtifactGOOS = runtime.GOOS

// ResolveArtifactPlatform returns the release artifact platform suffix for the
// running binary. It prefers the build-time ArtifactPlatform; when that is empty
// it falls back to runtime.GOARCH for the unambiguous architectures. For ARM
// without a build-time value it returns "" because the GOARM variant cannot be
// determined at runtime — callers MUST treat an empty result as "self-update not
// available on this build" rather than guessing a wrong artifact.
func ResolveArtifactPlatform() string {
	if ArtifactGOOS != "linux" {
		return ""
	}
	if ArtifactPlatform != "" {
		return ArtifactPlatform
	}
	switch runtime.GOARCH {
	case "amd64", "arm64", "386", "s390x":
		return runtime.GOARCH
	default:
		// arm (armv5/6/7) and anything else: not derivable without the build-time
		// value.
		return ""
	}
}
