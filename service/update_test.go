package service

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/config"
)

func TestVersionInfoFetchesAndCachesLatestRelease(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/deposist/s-ui-x/releases/tag/v9.9.9"}`))
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
			_, _ = w.Write([]byte(`{"tag_name":"v9.9.9","html_url":"https://github.com/deposist/s-ui-x/releases/tag/v9.9.9"}`))
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

func resetVersionCheckForTest(t *testing.T, client *http.Client, url string) {
	t.Helper()
	versionCheckState.Lock()
	oldClient := versionCheckState.client
	oldURL := versionCheckState.url
	oldCheckedAt := versionCheckState.checkedAt
	oldLatest := versionCheckState.latest
	oldReleaseURL := versionCheckState.releaseURL
	oldETag := versionCheckState.etag
	versionCheckState.client = client
	versionCheckState.url = url
	versionCheckState.checkedAt = time.Time{}
	versionCheckState.latest = ""
	versionCheckState.releaseURL = ""
	versionCheckState.etag = ""
	versionCheckState.Unlock()
	t.Cleanup(func() {
		versionCheckState.Lock()
		versionCheckState.client = oldClient
		versionCheckState.url = oldURL
		versionCheckState.checkedAt = oldCheckedAt
		versionCheckState.latest = oldLatest
		versionCheckState.releaseURL = oldReleaseURL
		versionCheckState.etag = oldETag
		versionCheckState.Unlock()
	})
}

func expireVersionCheckCacheForTest(t *testing.T) {
	t.Helper()
	versionCheckState.Lock()
	versionCheckState.checkedAt = time.Now().Add(-2 * versionCheckCache)
	versionCheckState.Unlock()
}
