package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/sagernet/sing-box/common/srs"
)

// Rule-set assets are .srs files the panel downloads itself and then hands to
// the core as `type: "local"`.
//
// Why this exists: a `type: "remote"` rule-set with an empty cache_file is
// fetched synchronously during startup, and a failed fetch aborts the whole
// core (route/rule/rule_set_remote.go). On a censored server the usual sources
// (raw.githubusercontent.com) are unreachable, so the core would refuse to
// start and take every proxy down with it. Materializing the files up-front
// makes startup independent of the network.
//
// The local path is not a free win: a missing or truncated local file is
// equally fatal (route/rule/rule_set_local.go). An earlier attempt in this
// project failed exactly that way ("no such file or directory") because the
// file was referenced before anything created it. Hence the contract below:
// download and verify everything first, and only report success when every
// file is on disk and parses.
const (
	// ruleSetMaxBytes caps a single download. The real sources are ~100-200 KB;
	// the cap only exists so a hostile or misconfigured URL cannot exhaust
	// panel memory.
	ruleSetMaxBytes = 32 << 20
	// ruleSetDownloadTimeout bounds one download attempt.
	ruleSetDownloadTimeout = 60 * time.Second
	// ruleSetDirName is the sub-directory of the DB folder holding the assets.
	ruleSetDirName = "rulesets"
	// ruleSetManifestName records which URL each materialized tag came from.
	//
	// It is required, not a convenience: a `type: "local"` rule-set stores only
	// a tag and a path, so once the config is converted the origin URL is gone.
	// Without this file a scheduled refresh would have nothing to re-download,
	// and the assets would silently freeze at their first downloaded version.
	ruleSetManifestName = "sources.json"
)

// ruleSetTagPattern constrains tags to characters that are safe in a filename.
// Tags arrive from API callers, so without this a tag like "../../etc/x" would
// let a caller choose where the panel writes.
var ruleSetTagPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// RuleSetSource is one requested download.
type RuleSetSource struct {
	Tag string `json:"tag"`
	URL string `json:"url"`
}

// RuleSetAsset is a materialized file, ready to be referenced from the config.
type RuleSetAsset struct {
	Tag  string `json:"tag"`
	Path string `json:"path"`
}

type RuleSetAssetService struct {
	SettingService
}

// RuleSetDir returns the directory holding materialized rule-set files.
func RuleSetDir() string {
	return filepath.Join(config.GetDBFolderPath(), ruleSetDirName)
}

// RuleSetAssetPath returns the on-disk path for a tag. The tag must already be
// validated by validateRuleSetTag.
func RuleSetAssetPath(tag string) string {
	return filepath.Join(RuleSetDir(), tag+".srs")
}

func validateRuleSetTag(tag string) error {
	if !ruleSetTagPattern.MatchString(tag) {
		return common.NewErrorf("invalid rule-set tag: %q", tag)
	}
	return nil
}

func validateRuleSetURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return common.NewErrorf("invalid rule-set url %q: %v", raw, err)
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return common.NewErrorf("rule-set url must be http or https: %q", raw)
	}
	if parsed.Host == "" {
		return common.NewErrorf("rule-set url has no host: %q", raw)
	}
	return nil
}

// GetRuleSetDownloadMode returns the configured download channel, falling back
// to "direct" so a missing or unreadable setting cannot make callers behave as
// if an outbound were configured.
func (s *SettingService) GetRuleSetDownloadMode() (string, error) {
	value, err := s.getString("ruleSetDownloadMode")
	if err != nil {
		return "direct", err
	}
	if strings.TrimSpace(value) == "outbound" {
		return "outbound", nil
	}
	return "direct", nil
}

// downloadClient builds the HTTP client for the configured channel. In
// "outbound" mode the download egresses through a running core outbound, which
// is what lets a censored server reach a blocked source through its own proxy.
func (s *RuleSetAssetService) downloadClient() (*http.Client, error) {
	mode, err := s.GetRuleSetDownloadMode()
	if err != nil {
		return nil, err
	}
	if mode != "outbound" {
		return &http.Client{Timeout: ruleSetDownloadTimeout}, nil
	}
	tag, err := s.getString("ruleSetDownloadOutbound")
	if err != nil {
		return nil, err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, common.NewError("rule-set download outbound is not configured")
	}
	return newCoreOutboundHTTPClient(tag, ruleSetDownloadTimeout)
}

// Materialize downloads every source, verifies it parses as a binary rule-set,
// and writes it to disk. It is all-or-nothing on purpose: nothing is written
// until every download has been fetched and verified, so a partial failure
// cannot leave the caller with a half-populated directory that it might then
// reference from the config.
//
// Existing files are left untouched when a refresh fails, so a working
// installation keeps its last-known-good assets.
func (s *RuleSetAssetService) Materialize(sources []RuleSetSource) ([]RuleSetAsset, error) {
	if len(sources) == 0 {
		return []RuleSetAsset{}, nil
	}
	seen := make(map[string]struct{}, len(sources))
	for _, src := range sources {
		if err := validateRuleSetTag(src.Tag); err != nil {
			return nil, err
		}
		if err := validateRuleSetURL(src.URL); err != nil {
			return nil, err
		}
		if _, dup := seen[src.Tag]; dup {
			return nil, common.NewErrorf("duplicate rule-set tag: %s", src.Tag)
		}
		seen[src.Tag] = struct{}{}
	}

	client, err := s.downloadClient()
	if err != nil {
		return nil, err
	}

	// Phase 1: fetch and verify everything in memory.
	payloads := make(map[string][]byte, len(sources))
	for _, src := range sources {
		data, err := fetchRuleSet(client, src.URL)
		if err != nil {
			return nil, common.NewErrorf("rule-set %s: %v", src.Tag, err)
		}
		payloads[src.Tag] = data
	}

	// Phase 2: only now touch the filesystem.
	dir := RuleSetDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, common.NewErrorf("cannot create rule-set directory: %v", err)
	}
	assets := make([]RuleSetAsset, 0, len(sources))
	for _, src := range sources {
		path := RuleSetAssetPath(src.Tag)
		if err := writeFileAtomic(path, payloads[src.Tag]); err != nil {
			return nil, common.NewErrorf("rule-set %s: %v", src.Tag, err)
		}
		assets = append(assets, RuleSetAsset{Tag: src.Tag, Path: path})
	}
	// Record where these files came from so a later refresh can update them.
	// A manifest failure must not fail the call: the .srs files are already on
	// disk and usable, and losing the ability to auto-refresh is far less
	// harmful than reporting a failure that makes the caller discard them.
	if err := mergeManifest(sources); err != nil {
		logger.Warning("rule-set manifest could not be written: ", err)
	}
	return assets, nil
}

// RuleSetManifestPath returns the path of the tag->url manifest.
func RuleSetManifestPath() string {
	return filepath.Join(RuleSetDir(), ruleSetManifestName)
}

// readManifest returns the recorded sources. A missing or unreadable manifest
// yields an empty map rather than an error: it only means there is nothing to
// refresh yet, which is a normal state before the first materialize.
func readManifest() map[string]string {
	data, err := os.ReadFile(RuleSetManifestPath())
	if err != nil {
		return map[string]string{}
	}
	var sources map[string]string
	if err := json.Unmarshal(data, &sources); err != nil {
		return map[string]string{}
	}
	return sources
}

// mergeManifest records the given sources, preserving entries for tags that
// were materialized by an earlier call and are not part of this one.
func mergeManifest(sources []RuleSetSource) error {
	merged := readManifest()
	for _, src := range sources {
		merged[src.Tag] = src.URL
	}
	data, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return writeFileAtomic(RuleSetManifestPath(), data)
}

// Refresh re-downloads every source recorded in the manifest.
//
// This replaces the `update_interval` that a remote rule-set would have used.
// Failure is deliberately non-fatal and leaves the existing files in place: the
// assets on disk are what keep the core startable, so a temporary network
// problem must never degrade them.
func (s *RuleSetAssetService) Refresh() (int, error) {
	recorded := readManifest()
	if len(recorded) == 0 {
		return 0, nil
	}
	sources := make([]RuleSetSource, 0, len(recorded))
	for tag, rawURL := range recorded {
		sources = append(sources, RuleSetSource{Tag: tag, URL: rawURL})
	}
	// Stable order keeps logs and tests deterministic.
	sort.Slice(sources, func(i, j int) bool { return sources[i].Tag < sources[j].Tag })

	assets, err := s.Materialize(sources)
	if err != nil {
		return 0, err
	}
	return len(assets), nil
}

func fetchRuleSet(client *http.Client, rawURL string) ([]byte, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, common.NewErrorf("download failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, common.NewErrorf("download failed with status %d", resp.StatusCode)
	}
	// Read one byte past the cap so an oversized body is detected rather than
	// silently truncated into a corrupt rule-set.
	data, err := io.ReadAll(io.LimitReader(resp.Body, ruleSetMaxBytes+1))
	if err != nil {
		return nil, common.NewErrorf("download failed: %v", err)
	}
	if len(data) > ruleSetMaxBytes {
		return nil, common.NewErrorf("rule-set is larger than %d bytes", ruleSetMaxBytes)
	}
	// Verifying here is what turns a captive-portal HTML page or a truncated
	// transfer into a clean error instead of a core that refuses to start.
	if err := verifyRuleSetBytes(data); err != nil {
		return nil, err
	}
	return data, nil
}

// verifyRuleSetBytes reports whether data is a rule-set the core can load.
func verifyRuleSetBytes(data []byte) error {
	if len(data) == 0 {
		return common.NewError("rule-set is empty")
	}
	if _, err := srs.Read(bytes.NewReader(data), false); err != nil {
		return common.NewErrorf("not a valid binary rule-set: %v", err)
	}
	return nil
}

// VerifyRuleSetFile reports whether a materialized file is present and loadable.
func VerifyRuleSetFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return verifyRuleSetBytes(data)
}

// writeFileAtomic writes via a temporary file and a rename, so a crash or a
// concurrent core start never observes a partially written rule-set.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		// No-op once the rename succeeded.
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	// Durability before the rename: otherwise a power loss can leave the
	// renamed file with no contents, which is fatal at core startup.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
