package service

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/util/common"
	"github.com/deposist/s-ui-x-extended/util/ssrf"

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
	// Bound a multi-source request even when every file is individually valid.
	ruleSetMaxBatchBytes = 64 << 20
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
var ruleSetAssetMu sync.Mutex
var ruleSetDirectTransportFactory = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = ssrf.NewPublicDialContext(net.DefaultResolver.LookupNetIP, (&net.Dialer{}).DialContext)
	return transport
}
var validateDirectRuleSetURL = func(ctx context.Context, rawURL string) error {
	return ssrf.ValidateOutboundURL(ctx, rawURL, "https")
}

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
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return common.NewErrorf("invalid rule-set url %q: %v", raw, err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return common.NewErrorf("rule-set url must be HTTPS: %q", raw)
	}
	if parsed.User != nil {
		return common.NewError("rule-set url must not include userinfo")
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
// downloadClient builds one immutable download plan. The mode is read exactly
// once, so validation and transport selection cannot diverge mid-request.
func (s *RuleSetAssetService) downloadClient(ctx context.Context) (*http.Client, bool, error) {
	mode, err := s.GetRuleSetDownloadMode()
	if err != nil {
		return nil, false, err
	}
	if mode != "outbound" {
		transport := ruleSetDirectTransportFactory()
		client := &http.Client{Timeout: ruleSetDownloadTimeout, Transport: transport}
		client.CheckRedirect = secureRuleSetRedirect(ctx, true)
		return client, true, nil
	}
	tag, err := s.getString("ruleSetDownloadOutbound")
	if err != nil {
		return nil, false, err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, false, common.NewError("rule-set download outbound is not configured")
	}
	client, err := newCoreOutboundHTTPClient(tag, ruleSetDownloadTimeout)
	if err != nil {
		return nil, false, err
	}
	client.CheckRedirect = secureRuleSetRedirect(ctx, false)
	return client, false, nil
}

func secureRuleSetRedirect(ctx context.Context, direct bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return common.NewError("too many rule-set redirects")
		}
		if err := validateRuleSetURL(req.URL.String()); err != nil {
			return err
		}
		if direct {
			return validateDirectRuleSetURL(ctx, req.URL.String())
		}
		return nil
	}
}

// Materialize downloads every source, verifies it parses as a binary rule-set,
// and writes it to disk. It is all-or-nothing on purpose: nothing is written
// until every download has been fetched and verified, so a partial failure
// cannot leave the caller with a half-populated directory that it might then
// reference from the config.
//
// Existing files are left untouched when a refresh fails, so a working
// installation keeps its last-known-good assets.
func (s *RuleSetAssetService) MaterializeContext(ctx context.Context, sources []RuleSetSource) ([]RuleSetAsset, error) {
	ruleSetAssetMu.Lock()
	defer ruleSetAssetMu.Unlock()
	return s.materializeLocked(ctx, sources)
}

func (s *RuleSetAssetService) materializeLocked(ctx context.Context, sources []RuleSetSource) ([]RuleSetAsset, error) {
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

	client, direct, err := s.downloadClient(ctx)
	if err != nil {
		return nil, err
	}
	if direct {
		for _, src := range sources {
			if err := validateDirectRuleSetURL(ctx, src.URL); err != nil {
				return nil, common.NewErrorf("rule-set %s: %v", src.Tag, err)
			}
		}
	}

	// Phase 1: fetch and verify everything in memory.
	payloads := make(map[string][]byte, len(sources))
	totalBytes := 0
	for _, src := range sources {
		data, err := fetchRuleSet(ctx, client, src.URL)
		if err != nil {
			return nil, common.NewErrorf("rule-set %s: %v", src.Tag, err)
		}
		totalBytes += len(data)
		if totalBytes > ruleSetMaxBatchBytes {
			return nil, common.NewErrorf("rule-set batch is larger than %d bytes", ruleSetMaxBatchBytes)
		}
		payloads[src.Tag] = data
	}

	return commitRuleSetBatch(sources, payloads)
}

type stagedRuleSetFile struct {
	source     RuleSetSource
	label      string
	path       string
	temporary  string
	backup     string
	hadCurrent bool
}

// commitRuleSetBatch stages every file first, then replaces the live batch. If
// a later replacement or the manifest commit fails, prior files are restored.
func commitRuleSetBatch(sources []RuleSetSource, payloads map[string][]byte) ([]RuleSetAsset, error) {
	dir := RuleSetDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, common.NewErrorf("cannot create rule-set directory: %v", err)
	}
	mergedManifest := readManifest()
	for _, src := range sources {
		mergedManifest[src.Tag] = src.URL
	}
	manifestData, err := json.Marshal(mergedManifest)
	if err != nil {
		return nil, err
	}
	staged := make([]stagedRuleSetFile, 0, len(sources)+1)
	cleanup := func() {
		for _, file := range staged {
			if file.temporary != "" {
				_ = os.Remove(file.temporary)
			}
			if file.backup != "" {
				_ = os.Remove(file.backup)
			}
		}
	}
	defer cleanup()

	for _, src := range sources {
		path := RuleSetAssetPath(src.Tag)
		temporary, err := stageRuleSetFile(path, payloads[src.Tag])
		if err != nil {
			return nil, common.NewErrorf("rule-set %s: %v", src.Tag, err)
		}
		staged = append(staged, stagedRuleSetFile{source: src, label: src.Tag, path: path, temporary: temporary})
	}
	manifestTemporary, err := stageRuleSetFile(RuleSetManifestPath(), manifestData)
	if err != nil {
		return nil, common.NewErrorf("rule-set manifest: %v", err)
	}
	staged = append(staged, stagedRuleSetFile{label: "manifest", path: RuleSetManifestPath(), temporary: manifestTemporary})

	rollback := func(committed int) {
		for i := committed - 1; i >= 0; i-- {
			file := &staged[i]
			_ = os.Remove(file.path)
			if file.hadCurrent {
				_ = os.Rename(file.backup, file.path)
				file.backup = ""
			}
		}
	}
	for i := range staged {
		file := &staged[i]
		if _, err := os.Stat(file.path); err == nil {
			backup, err := reserveBackupName(file.path)
			if err != nil {
				rollback(i)
				return nil, err
			}
			if err := os.Rename(file.path, backup); err != nil {
				rollback(i)
				return nil, err
			}
			file.backup, file.hadCurrent = backup, true
		} else if !os.IsNotExist(err) {
			rollback(i)
			return nil, err
		}
		if err := os.Rename(file.temporary, file.path); err != nil {
			if file.hadCurrent {
				_ = os.Rename(file.backup, file.path)
				file.backup = ""
			}
			rollback(i)
			return nil, common.NewErrorf("rule-set %s: %v", file.label, err)
		}
		file.temporary = ""
	}
	if err := syncDirectory(dir); err != nil {
		rollback(len(staged))
		return nil, err
	}
	assets := make([]RuleSetAsset, 0, len(sources))
	for i := range staged {
		if i < len(sources) {
			assets = append(assets, RuleSetAsset{Tag: staged[i].source.Tag, Path: staged[i].path})
		}
		if staged[i].backup != "" {
			_ = os.Remove(staged[i].backup)
			staged[i].backup = ""
		}
	}
	return assets, nil
}

func stageRuleSetFile(path string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(name, 0o640)
	}
	if err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func reserveBackupName(path string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.bak")
	if err != nil {
		return "", err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	// Windows does not expose directory FlushFileBuffers through os.File.Sync.
	if runtime.GOOS == "windows" {
		err = nil
	}
	if err != nil {
		return err
	}
	return closeErr
}

func (s *RuleSetAssetService) Materialize(sources []RuleSetSource) ([]RuleSetAsset, error) {
	return s.MaterializeContext(context.Background(), sources)
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

// Refresh re-downloads every source recorded in the manifest.
//
// This replaces the `update_interval` that a remote rule-set would have used.
// Failure is deliberately non-fatal and leaves the existing files in place: the
// assets on disk are what keep the core startable, so a temporary network
// problem must never degrade them.
func (s *RuleSetAssetService) RefreshContext(ctx context.Context) (int, error) {
	ruleSetAssetMu.Lock()
	defer ruleSetAssetMu.Unlock()
	recorded := readManifest()
	if len(recorded) == 0 {
		return 0, nil
	}
	sources := make([]RuleSetSource, 0, len(recorded))
	for tag, rawURL := range recorded {
		sources = append(sources, RuleSetSource{Tag: tag, URL: rawURL})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Tag < sources[j].Tag })

	assets, err := s.materializeLocked(ctx, sources)
	if err != nil {
		return 0, err
	}
	return len(assets), nil
}

func (s *RuleSetAssetService) Refresh() (int, error) {
	return s.RefreshContext(context.Background())
}

func fetchRuleSet(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, common.NewErrorf("download failed: %v", err)
	}
	resp, err := client.Do(request)
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

// verifyRuleSetBytes reports whether data is a complete rule-set the core can
// load. srs.Read stops after the declared rules; drain the zlib stream as a
// second pass so a corrupt or missing checksum trailer is not accepted.
func verifyRuleSetBytes(data []byte) error {
	if len(data) == 0 {
		return common.NewError("rule-set is empty")
	}
	if _, err := srs.Read(bytes.NewReader(data), false); err != nil {
		return common.NewErrorf("not a valid binary rule-set: %v", err)
	}
	if len(data) < 5 {
		return common.NewError("not a complete binary rule-set")
	}
	reader, err := zlib.NewReader(bytes.NewReader(data[4:]))
	if err != nil {
		return common.NewErrorf("not a valid binary rule-set: %v", err)
	}
	_, readErr := io.Copy(io.Discard, reader)
	closeErr := reader.Close()
	if readErr != nil {
		return common.NewErrorf("not a complete binary rule-set: %v", readErr)
	}
	if closeErr != nil {
		return common.NewErrorf("not a complete binary rule-set: %v", closeErr)
	}
	return nil
}

// VerifyRuleSetFile reports whether a materialized file is a bounded, regular,
// loadable rule-set. Config paths are caller-controlled, so devices, FIFOs and
// oversized files must be rejected before reading them into memory.
func VerifyRuleSetFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return common.NewError("rule-set path is not a regular file")
	}
	if info.Size() > ruleSetMaxBytes {
		return common.NewErrorf("rule-set is larger than %d bytes", ruleSetMaxBytes)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, ruleSetMaxBytes+1))
	if err != nil {
		return err
	}
	if len(data) > ruleSetMaxBytes {
		return common.NewErrorf("rule-set is larger than %d bytes", ruleSetMaxBytes)
	}
	return verifyRuleSetBytes(data)
}

// writeFileAtomic writes via a temporary file and a rename, so a crash or a
// concurrent core start never observes a partially written rule-set.
// writeFileAtomic stages, renames, and syncs the containing directory so the
// replacement survives a power loss after the function reports success.
func writeFileAtomic(path string, data []byte) error {
	temporary, err := stageRuleSetFile(path, data)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
