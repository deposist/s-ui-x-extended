package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/deposist/s-ui-x-extended/config"
)

// httpDoer is the minimal HTTP surface used by the version checker, so tests can
// inject a transport. The real client (defaultVersionCheckClient) keeps Go's
// default TLS verification - it MUST NOT be disabled (SR-003).
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func defaultVersionCheckClient() *http.Client {
	return &http.Client{Timeout: versionCheckTimeout}
}

var (
	errNoRelease = errors.New("no release found for channel")
	errNotNewer  = errors.New("installed version is up to date")
	errNoAsset   = errors.New("no installable artifact for this platform")
)

func errUpdateCheck(msg string) error { return fmt.Errorf("update check: %s", msg) }

type ghAsset struct {
	Name string `json:"name"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	HTMLURL    string    `json:"html_url"`
	Body       string    `json:"body"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

// fetchChannelRelease queries GitHub for the channel's target release. main uses
// /releases/latest (GitHub excludes pre-releases); beta lists releases and picks
// the highest semver (so a freshly-published stable supersedes a beta - the
// "graduation" case). Returns (release, etag, notModified, err).
func fetchChannelRelease(client httpDoer, base string, channel string, etag string) (*resolvedRelease, string, bool, error) {
	url := base + "/releases/latest"
	if channel == config.UpdateChannelBeta {
		url = base + "/releases?per_page=20"
	}
	body, respETag, notModified, err := doReleaseRequest(client, url, etag)
	if err != nil || notModified {
		return nil, respETag, notModified, err
	}
	if channel == config.UpdateChannelBeta {
		var releases []ghRelease
		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, "", false, err
		}
		return resolveRelease(selectBetaRelease(releases)), respETag, false, nil
	}
	var release ghRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, "", false, err
	}
	return resolveRelease(&release), respETag, false, nil
}

func doReleaseRequest(client httpDoer, url string, etag string) ([]byte, string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "s-ui-version-check")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	respETag := strings.TrimSpace(resp.Header.Get("ETag"))
	if resp.StatusCode == http.StatusNotModified {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		if respETag == "" {
			respETag = etag
		}
		return nil, respETag, true, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, "", false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, "", false, err
	}
	return body, respETag, false, nil
}

// selectBetaRelease returns the highest-semver non-draft release (stable + beta).
func selectBetaRelease(releases []ghRelease) *ghRelease {
	var best *ghRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || strings.TrimSpace(r.TagName) == "" {
			continue
		}
		if best == nil {
			best = r
			continue
		}
		if cmp, ok := config.CompareVersions(r.TagName, best.TagName); ok && cmp > 0 {
			best = r
		}
	}
	return best
}

// resolveRelease builds the cached release view, computing the artifact URLs from
// a fixed template (SR-004) and whether an installable asset exists for this
// platform.
func resolveRelease(release *ghRelease) *resolvedRelease {
	if release == nil {
		return nil
	}
	tag := strings.TrimSpace(release.TagName)
	if tag == "" {
		return nil
	}
	platform := config.ResolveArtifactPlatform()
	resolved := &resolvedRelease{
		tag:        tag,
		version:    config.NormalizeVersion(tag),
		prerelease: release.Prerelease || isPrereleaseTag(tag),
		notes:      strings.TrimSpace(release.Body),
		htmlURL:    strings.TrimSpace(release.HTMLURL),
		platform:   platform,
	}
	if platform != "" {
		assetName := fmt.Sprintf("s-ui-linux-%s.tar.gz", platform)
		resolved.assetURL = fmt.Sprintf("%s/%s/%s", githubDownloadBase, tag, assetName)
		resolved.checksumURL = resolved.assetURL + ".sha256"
		resolved.assetAvailable = hasAsset(release.Assets, assetName) && hasAsset(release.Assets, assetName+".sha256")
	}
	return resolved
}

func hasAsset(assets []ghAsset, name string) bool {
	for _, a := range assets {
		if a.Name == name {
			return true
		}
	}
	return false
}

// isPrereleaseTag classifies a tag as a pre-release by its semver suffix
// (e.g. 1.5.9-beta2), independent of GitHub's prerelease flag.
func isPrereleaseTag(tag string) bool {
	semver, ok := config.ParseSemver(strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V"))
	return ok && len(semver.Prerelease) > 0
}
