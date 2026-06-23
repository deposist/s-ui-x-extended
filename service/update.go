package service

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
)

const (
	githubAPIBase       = "https://api.github.com/repos/deposist/s-ui-x-extended"
	githubDownloadBase  = "https://github.com/deposist/s-ui-x-extended/releases/download"
	versionCheckCache   = time.Hour
	versionCheckTimeout = 3 * time.Second
)

type VersionService struct{}

type VersionInfo struct {
	Current         string `json:"current"`
	Version         string `json:"version"`
	Channel         string `json:"channel,omitempty"`
	Latest          string `json:"latest,omitempty"`
	Prerelease      bool   `json:"prerelease,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable,omitempty"`
	AssetAvailable  bool   `json:"assetAvailable,omitempty"`
	ReleaseURL      string `json:"releaseURL,omitempty"`
	ReleaseNotes    string `json:"releaseNotes,omitempty"`
	CheckedAt       int64  `json:"checkedAt,omitempty"`
	CheckError      string `json:"checkError,omitempty"`
}

// ReleaseTarget is a concrete release selected for a channel, consumed by the
// panel self-update apply path. URLs are built from a fixed template (SR-004).
type ReleaseTarget struct {
	Channel      string
	Tag          string
	Version      string
	Prerelease   bool
	ReleaseNotes string
	ReleaseURL   string
	Platform     string
	AssetURL     string
	ChecksumURL  string
}

// resolvedRelease is the cached, channel-specific view of a GitHub release.
type resolvedRelease struct {
	tag            string
	version        string
	prerelease     bool
	notes          string
	htmlURL        string
	platform       string
	assetURL       string
	checksumURL    string
	assetAvailable bool
}

type channelState struct {
	checkedAt time.Time
	etag      string
	release   *resolvedRelease
	lastErr   string
}

var versionCheckState = struct {
	sync.Mutex
	client   httpDoer
	baseURL  string
	channels map[string]*channelState
}{
	client:   defaultVersionCheckClient(),
	baseURL:  githubAPIBase,
	channels: map[string]*channelState{},
}

func init() {
	database.RegisterResetHook("service.version_check", resetVersionCheckCache)
}

func resetVersionCheckCache() {
	versionCheckState.Lock()
	defer versionCheckState.Unlock()
	versionCheckState.channels = map[string]*channelState{}
}

// GetVersionInfo reports the stable ("main") channel version status. Kept
// channel-agnostic for the existing /api/version consumers (backward compatible).
func (s *VersionService) GetVersionInfo() VersionInfo {
	return s.CheckForChannel(config.UpdateChannelMain, false)
}

// CheckForChannel returns version status for a channel. force bypasses the cache
// (used by the explicit "Check updates" action).
func (s *VersionService) CheckForChannel(channel string, force bool) VersionInfo {
	channel = config.NormalizeUpdateChannel(channel)
	current := config.GetVersion()
	info := VersionInfo{Current: current, Version: current, Channel: channel}

	release, checkedAt, errStr := cachedReleaseForChannel(channel, force)
	if !checkedAt.IsZero() {
		info.CheckedAt = checkedAt.Unix()
	}
	if errStr != "" {
		info.CheckError = errStr
	}
	if release == nil {
		return info
	}
	info.Latest = release.tag
	info.Prerelease = release.prerelease
	info.ReleaseURL = release.htmlURL
	info.ReleaseNotes = release.notes
	info.AssetAvailable = release.assetAvailable
	info.UpdateAvailable = versionIsNewer(release.tag, current)
	return info
}

// ResolveTarget returns the concrete release to update to on a channel, or an
// error if there is nothing newer or no installable artifact for this platform.
func (s *VersionService) ResolveTarget(channel string) (ReleaseTarget, error) {
	channel = config.NormalizeUpdateChannel(channel)
	release, _, errStr := cachedReleaseForChannel(channel, true)
	if release == nil {
		if errStr != "" {
			return ReleaseTarget{}, errUpdateCheck(errStr)
		}
		return ReleaseTarget{}, errNoRelease
	}
	current := config.GetVersion()
	if !versionIsNewer(release.tag, current) {
		return ReleaseTarget{}, errNotNewer
	}
	if !release.assetAvailable {
		return ReleaseTarget{}, errNoAsset
	}
	return ReleaseTarget{
		Channel:      channel,
		Tag:          release.tag,
		Version:      release.version,
		Prerelease:   release.prerelease,
		ReleaseNotes: release.notes,
		ReleaseURL:   release.htmlURL,
		Platform:     release.platform,
		AssetURL:     release.assetURL,
		ChecksumURL:  release.checksumURL,
	}, nil
}

func cachedReleaseForChannel(channel string, force bool) (*resolvedRelease, time.Time, string) {
	versionCheckState.Lock()
	st := channelStateLocked(channel)
	now := time.Now()
	if !force && !st.checkedAt.IsZero() && now.Sub(st.checkedAt) < versionCheckCache {
		release, at, errStr := st.release, st.checkedAt, st.lastErr
		versionCheckState.Unlock()
		return release, at, errStr
	}
	client := versionCheckState.client
	base := versionCheckState.baseURL
	etag := st.etag
	versionCheckState.Unlock()

	release, respETag, notModified, err := fetchChannelRelease(client, base, channel, etag)

	versionCheckState.Lock()
	defer versionCheckState.Unlock()
	st = channelStateLocked(channel)
	st.checkedAt = now
	if err != nil {
		logger.Warning("version check failed:", err)
		st.lastErr = "version check failed"
		return st.release, st.checkedAt, st.lastErr
	}
	st.lastErr = ""
	if notModified {
		if respETag != "" {
			st.etag = respETag
		}
		return st.release, st.checkedAt, ""
	}
	st.etag = respETag
	st.release = release
	return st.release, st.checkedAt, ""
}

func channelStateLocked(channel string) *channelState {
	st, ok := versionCheckState.channels[channel]
	if !ok {
		st = &channelState{}
		versionCheckState.channels[channel] = st
	}
	return st
}

func versionIsNewer(latest string, current string) bool {
	return config.VersionIsNewer(latest, current)
}
