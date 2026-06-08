package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
)

const (
	versionCheckURL     = "https://api.github.com/repos/deposist/s-ui-x/releases/latest"
	versionCheckCache   = time.Hour
	versionCheckTimeout = 3 * time.Second
)

type VersionService struct{}

type VersionInfo struct {
	Current         string `json:"current"`
	Version         string `json:"version"`
	Latest          string `json:"latest,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable,omitempty"`
	ReleaseURL      string `json:"releaseURL,omitempty"`
	CheckedAt       int64  `json:"checkedAt,omitempty"`
}

type latestRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

var versionCheckState = struct {
	sync.Mutex
	client     *http.Client
	url        string
	checkedAt  time.Time
	latest     string
	releaseURL string
	etag       string
}{
	client: &http.Client{Timeout: versionCheckTimeout},
	url:    versionCheckURL,
}

func init() {
	database.RegisterResetHook("service.version_check", resetVersionCheckCache)
}

func resetVersionCheckCache() {
	versionCheckState.Lock()
	defer versionCheckState.Unlock()
	versionCheckState.checkedAt = time.Time{}
	versionCheckState.latest = ""
	versionCheckState.releaseURL = ""
	versionCheckState.etag = ""
}

func (s *VersionService) GetVersionInfo() VersionInfo {
	current := config.GetVersion()
	latest, releaseURL, checkedAt := latestReleaseCached()
	info := VersionInfo{
		Current: current,
		Version: current,
	}
	if latest == "" {
		return info
	}
	info.Latest = latest
	info.ReleaseURL = releaseURL
	info.CheckedAt = checkedAt.Unix()
	info.UpdateAvailable = versionIsNewer(latest, current)
	return info
}

func latestReleaseCached() (string, string, time.Time) {
	versionCheckState.Lock()
	now := time.Now()
	if !versionCheckState.checkedAt.IsZero() && now.Sub(versionCheckState.checkedAt) < versionCheckCache {
		latest := versionCheckState.latest
		releaseURL := versionCheckState.releaseURL
		checkedAt := versionCheckState.checkedAt
		versionCheckState.Unlock()
		return latest, releaseURL, checkedAt
	}
	client := versionCheckState.client
	url := versionCheckState.url
	etag := versionCheckState.etag
	versionCheckState.Unlock()

	latest, releaseURL, responseETag, notModified, err := fetchLatestRelease(client, url, etag)
	if err != nil {
		logger.Warning("version check failed:", err)
	}

	versionCheckState.Lock()
	defer versionCheckState.Unlock()
	versionCheckState.checkedAt = now
	if err == nil && notModified {
		versionCheckState.etag = responseETag
	} else if err == nil {
		versionCheckState.latest = latest
		versionCheckState.releaseURL = releaseURL
		versionCheckState.etag = responseETag
	}
	return versionCheckState.latest, versionCheckState.releaseURL, versionCheckState.checkedAt
}

func fetchLatestRelease(client *http.Client, url string, etag string) (string, string, string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "s-ui-version-check")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", false, err
	}
	defer resp.Body.Close()
	responseETag := strings.TrimSpace(resp.Header.Get("ETag"))
	if resp.StatusCode == http.StatusNotModified {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		if responseETag == "" {
			responseETag = etag
		}
		return "", "", responseETag, true, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return "", "", "", false, http.ErrNotSupported
	}
	var release latestRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", "", "", false, err
	}
	release.TagName = strings.TrimSpace(release.TagName)
	release.HTMLURL = strings.TrimSpace(release.HTMLURL)
	return release.TagName, release.HTMLURL, responseETag, false, nil
}

func versionIsNewer(latest string, current string) bool {
	return config.VersionIsNewer(latest, current)
}
