package service

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
)

func TestVersionUpdateEndpointsUseExtendedRepository(t *testing.T) {
	if githubAPIBase != "https://api.github.com/repos/deposist/s-ui-x-extended" {
		t.Fatalf("githubAPIBase = %q", githubAPIBase)
	}
	if githubDownloadBase != "https://github.com/deposist/s-ui-x-extended/releases/download" {
		t.Fatalf("githubDownloadBase = %q", githubDownloadBase)
	}
}

func TestVersionInfoFetchesAndCachesLatestRelease(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/deposist/s-ui-x-extended/releases/tag/v9.9.9"}`))
	}))
	defer server.Close()
	resetVersionCheckForTest(t, server.Client(), server.URL)

	info := (&VersionService{}).GetVersionInfo()
	if info.Current != config.GetVersion() || info.Version != config.GetVersion() {
		t.Fatalf("current version missing: %#v", info)
	}
	if info.Latest != "v9.9.9" || !info.UpdateAvailable || info.ReleaseURL == "" || info.CheckedAt == 0 {
		t.Fatalf("latest release not populated: %#v", info)
	}
	_ = (&VersionService{}).GetVersionInfo()
	if calls.Load() != 1 {
		t.Fatalf("version cache was not used, calls=%d", calls.Load())
	}
}

func TestVersionInfoFailsSoftAndCachesFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	resetVersionCheckForTest(t, server.Client(), server.URL)

	info := (&VersionService{}).GetVersionInfo()
	if info.Current != config.GetVersion() || info.Latest != "" || info.UpdateAvailable {
		t.Fatalf("version check should fail soft: %#v", info)
	}
	if info.CheckError == "" {
		t.Fatalf("soft failure should surface a checkError: %#v", info)
	}
	_ = (&VersionService{}).GetVersionInfo()
	if calls.Load() != 1 {
		t.Fatalf("failed version check should be cached, calls=%d", calls.Load())
	}
}

func TestVersionInfoUsesETagAfterCacheExpiryIssue29(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch call := calls.Add(1); call {
		case 1:
			if got := r.Header.Get("If-None-Match"); got != "" {
				t.Fatalf("first request sent If-None-Match=%q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("ETag", `"release-v1"`)
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/deposist/s-ui-x-extended/releases/tag/v9.9.9"}`))
		case 2:
			if got, want := r.Header.Get("If-None-Match"), `"release-v1"`; got != want {
				t.Fatalf("If-None-Match=%q, want %q", got, want)
			}
			w.WriteHeader(http.StatusNotModified)
		default:
			t.Fatalf("unexpected release request #%d", call)
		}
	}))
	defer server.Close()
	resetVersionCheckForTest(t, server.Client(), server.URL)

	first := (&VersionService{}).GetVersionInfo()
	if first.Latest != "v9.9.9" || first.ReleaseURL == "" || first.CheckedAt == 0 {
		t.Fatalf("latest release not populated: %#v", first)
	}
	expireVersionCheckCacheForTest(t)

	second := (&VersionService{}).GetVersionInfo()
	if second.Latest != first.Latest || second.ReleaseURL != first.ReleaseURL {
		t.Fatalf("304 should preserve cached release, first=%#v second=%#v", first, second)
	}
	if calls.Load() != 2 {
		t.Fatalf("expired cache should make exactly two requests, calls=%d", calls.Load())
	}

	_ = (&VersionService{}).GetVersionInfo()
	if calls.Load() != 2 {
		t.Fatalf("fresh 304 cache should avoid a third request, calls=%d", calls.Load())
	}
}

func TestVersionIsNewer(t *testing.T) {
	if !versionIsNewer("v1.6.0", "1.5.0") {
		t.Fatal("expected v1.6.0 to be newer than 1.5.0")
	}
	if versionIsNewer("v1.4.9", "1.5.0") {
		t.Fatal("older version detected as newer")
	}
	if versionIsNewer("v1.5.2-beta.1", "1.5.2") {
		t.Fatal("prerelease detected as newer than final release")
	}
}

// T010: beta channel picks the highest semver including a freshly-published
// stable, i.e. graduation beta -> stable (1.5.9-beta2 -> 1.5.9).
func TestSelectBetaReleasePicksHighestIncludingStableGraduation(t *testing.T) {
	releases := []ghRelease{
		{TagName: "v1.5.8"},
		{TagName: "v1.5.9-beta2"},
		{TagName: "v1.5.9"}, // stable graduation of the beta
		{TagName: "v2.0.0", Draft: true},
	}
	best := selectBetaRelease(releases)
	if best == nil || best.TagName != "v1.5.9" {
		t.Fatalf("beta channel should pick stable v1.5.9 over beta and skip drafts, got %#v", best)
	}
}

// T010: artifact URLs are derived from a fixed template (SR-004) and asset
// availability is taken from the release's published assets.
func TestResolveReleaseBuildsAssetURLsFromTemplate(t *testing.T) {
	setArtifactPlatformForTest(t, "amd64")
	resolved := resolveRelease(&ghRelease{
		TagName: "v9.9.9",
		Body:    "release notes here",
		Assets: []ghAsset{
			{Name: "s-ui-linux-amd64.tar.gz"},
			{Name: "s-ui-linux-amd64.tar.gz.sha256"},
		},
	})
	if resolved == nil || !resolved.assetAvailable {
		t.Fatalf("expected installable asset, got %#v", resolved)
	}
	wantAsset := "https://github.com/deposist/s-ui-x-extended/releases/download/v9.9.9/s-ui-linux-amd64.tar.gz"
	if resolved.assetURL != wantAsset || resolved.checksumURL != wantAsset+".sha256" {
		t.Fatalf("artifact URLs not from template: %#v", resolved)
	}
	if resolved.notes != "release notes here" {
		t.Fatalf("release notes not captured: %#v", resolved)
	}
}

// T010: beta channel over HTTP surfaces the graduated stable as the latest.
func TestCheckForChannelBetaGraduationOverHTTP(t *testing.T) {
	setArtifactPlatformForTest(t, "amd64")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"tag_name":"v99.0.0-beta1","prerelease":true,"assets":[{"name":"s-ui-linux-amd64.tar.gz"},{"name":"s-ui-linux-amd64.tar.gz.sha256"}]},
			{"tag_name":"v99.0.0","prerelease":false,"assets":[{"name":"s-ui-linux-amd64.tar.gz"},{"name":"s-ui-linux-amd64.tar.gz.sha256"}]}
		]`))
	}))
	defer server.Close()
	resetVersionCheckForTest(t, server.Client(), server.URL)

	info := (&VersionService{}).CheckForChannel(config.UpdateChannelBeta, true)
	if info.Channel != config.UpdateChannelBeta {
		t.Fatalf("channel not echoed: %#v", info)
	}
	if info.Latest != "v99.0.0" || info.Prerelease {
		t.Fatalf("beta channel should graduate to stable v99.0.0, got %#v", info)
	}
	if !info.UpdateAvailable || !info.AssetAvailable {
		t.Fatalf("expected an installable update, got %#v", info)
	}
}

// T010: downgrade guard and missing-asset guard for the apply target.
func TestResolveTargetGuards(t *testing.T) {
	setArtifactPlatformForTest(t, "amd64")

	// Older-than-current release on the channel -> no downgrade.
	older := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.0.1","assets":[{"name":"s-ui-linux-amd64.tar.gz"},{"name":"s-ui-linux-amd64.tar.gz.sha256"}]}`))
	}))
	defer older.Close()
	resetVersionCheckForTest(t, older.Client(), older.URL)
	if _, err := (&VersionService{}).ResolveTarget(config.UpdateChannelMain); err != errNotNewer {
		t.Fatalf("expected errNotNewer for downgrade, got %v", err)
	}

	// Newer release but no asset for this platform -> not installable.
	noAsset := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","assets":[]}`))
	}))
	defer noAsset.Close()
	resetVersionCheckForTest(t, noAsset.Client(), noAsset.URL)
	if _, err := (&VersionService{}).ResolveTarget(config.UpdateChannelMain); err != errNoAsset {
		t.Fatalf("expected errNoAsset, got %v", err)
	}
}

func resetVersionCheckForTest(t *testing.T, client httpDoer, baseURL string) {
	t.Helper()
	versionCheckState.Lock()
	oldClient := versionCheckState.client
	oldBase := versionCheckState.baseURL
	oldChannels := versionCheckState.channels
	versionCheckState.client = client
	versionCheckState.baseURL = baseURL
	versionCheckState.channels = map[string]*channelState{}
	versionCheckState.Unlock()
	t.Cleanup(func() {
		versionCheckState.Lock()
		versionCheckState.client = oldClient
		versionCheckState.baseURL = oldBase
		versionCheckState.channels = oldChannels
		versionCheckState.Unlock()
	})
}

func expireVersionCheckCacheForTest(t *testing.T) {
	t.Helper()
	versionCheckState.Lock()
	for _, st := range versionCheckState.channels {
		st.checkedAt = time.Now().Add(-2 * versionCheckCache)
	}
	versionCheckState.Unlock()
}

func setArtifactPlatformForTest(t *testing.T, platform string) {
	t.Helper()
	old := config.ArtifactPlatform
	config.ArtifactPlatform = platform
	t.Cleanup(func() { config.ArtifactPlatform = old })
}
