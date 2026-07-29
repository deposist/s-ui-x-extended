package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/logger"
)

const (
	backupSuffix     = ".bak"
	pendingSuffix    = ".update-pending"
	maxArtifactBytes = 512 << 20 // 512 MiB ceiling for the release tarball
	maxChecksumBytes = 4 << 10
	downloadTimeout  = 5 * time.Minute
	// rollbackAfterAttempts is how many failed boots of a freshly-applied binary
	// trigger an automatic restore of the backup (SR-012).
	rollbackAfterAttempts = 2
)

var (
	errChecksumMismatch      = errors.New("artifact checksum does not match the published value")
	errArtifactTooLarge      = errors.New("artifact exceeds the size limit")
	errChecksumTooLarge      = errors.New("checksum file exceeds the size limit")
	errArchiveMemberTooLarge = errors.New("archive member exceeds the size limit")
	errManifestInvalid       = errors.New("update manifest is invalid")
)

const maxManifestBytes = 16 << 10

type signedUpdateManifest struct {
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Platform string `json:"platform"`
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
}

type panelUpdateDeps struct {
	client   httpDoer
	execPath string
}

func defaultPanelUpdateDeps() panelUpdateDeps {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	return panelUpdateDeps{
		client:   &http.Client{Timeout: downloadTimeout},
		execPath: exe,
	}
}

// applyPipeline downloads, integrity-checks, extracts and atomically swaps the
// panel binary. It only mutates the live executable at the final rename, and
// only after a successful checksum verification (SR-002, SR-007).
func applyPipeline(target ReleaseTarget, deps panelUpdateDeps, setStage func(UpdateStage)) (bool, error) {
	if deps.execPath == "" {
		return false, errors.New("cannot locate current executable")
	}
	dir := filepath.Dir(deps.execPath)

	setStage(UpdateStageDownloading)
	archive := filepath.Join(dir, ".sui-update.tar.gz")
	defer func() {
		if err := removeUpdateFile(archive); err != nil {
			logger.Warning("panel update: could not remove downloaded archive:", err)
		}
	}()
	if err := downloadToFile(deps.client, target.AssetURL, archive); err != nil {
		return false, err
	}
	expected, err := downloadChecksum(deps.client, target.ChecksumURL)
	if err != nil {
		return false, err
	}
	manifest, err := downloadUpdateManifest(deps.client, target.AssetURL+".manifest.json")
	if err != nil {
		return false, err
	}
	setStage(UpdateStageVerifying)
	if err := verifyUpdateManifest(manifest, target, expected); err != nil {
		return false, err
	}
	if err := verifySHA256(archive, expected); err != nil {
		return false, err
	}
	setStage(UpdateStageApplying)
	if err := swapBinary(archive, deps.execPath); err != nil {
		return false, err
	}
	return true, nil
}

// swapBinary extracts the new binary next to the current one and atomically
// replaces it, keeping the previous binary as <exec>.bak for rollback.
func swapBinary(archive string, execPath string) error {
	newBin := execPath + ".new"
	if err := extractBinary(archive, newBin); err != nil {
		return errors.Join(err, removeUpdateFile(newBin))
	}
	// #nosec G302 -- the staged program must be executable before the atomic swap.
	if err := os.Chmod(newBin, 0o755); err != nil {
		return errors.Join(err, removeUpdateFile(newBin))
	}
	if err := copyFile(execPath, execPath+backupSuffix); err != nil {
		return errors.Join(err, removeUpdateFile(newBin))
	}
	if err := os.Rename(newBin, execPath); err != nil {
		return errors.Join(err, removeUpdateFile(newBin)) // live binary is untouched; .bak remains for safety
	}
	return nil
}

func downloadToFile(client httpDoer, url string, dest string) error {
	return downloadToFileLimit(client, url, dest, maxArtifactBytes)
}

func downloadToFileLimit(client httpDoer, url string, dest string, limit int64) error {
	if limit < 0 {
		return errArtifactTooLarge
	}
	if !strings.HasPrefix(url, "https://") { // SR-003/SR-004: TLS-only, template URLs
		return fmt.Errorf("refusing non-https artifact url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "s-ui-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("artifact download failed: status %d", resp.StatusCode)
	}
	// #nosec G304 -- dest is a fixed filename next to the running executable.
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(f, io.LimitReader(resp.Body, sizeProbeLimit(limit)))
	if copyErr != nil {
		return errors.Join(copyErr, f.Close(), removeUpdateFile(dest))
	}
	if written > limit {
		return errors.Join(errArtifactTooLarge, f.Close(), removeUpdateFile(dest))
	}
	if err := f.Sync(); err != nil {
		return errors.Join(err, f.Close(), removeUpdateFile(dest))
	}
	if err := f.Close(); err != nil {
		return errors.Join(err, removeUpdateFile(dest))
	}
	return nil
}

func downloadChecksum(client httpDoer, url string) (string, error) {
	return downloadChecksumLimit(client, url, maxChecksumBytes)
}

func downloadChecksumLimit(client httpDoer, url string, limit int64) (string, error) {
	if limit < 0 {
		return "", errChecksumTooLarge
	}
	if !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("refusing non-https checksum url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "s-ui-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("checksum download failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, sizeProbeLimit(limit)))
	if err != nil {
		return "", err
	}
	if int64(len(body)) > limit {
		return "", errChecksumTooLarge
	}
	// Format produced by `sha256sum`: "<hex>  <filename>".
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}
	return strings.ToLower(fields[0]), nil
}

func downloadUpdateManifest(client httpDoer, url string) ([]byte, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("refusing non-https manifest url")
	}
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "s-ui-self-update")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest download failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxManifestBytes {
		return nil, errManifestInvalid
	}
	return body, nil
}

func verifyUpdateManifest(rawManifest []byte, target ReleaseTarget, checksum string) error {
	var manifest signedUpdateManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return errors.Join(errManifestInvalid, err)
	}
	filename := fmt.Sprintf("s-ui-linux-%s.tar.gz", target.Platform)
	if manifest.Version != target.Version || manifest.Channel != target.Channel ||
		manifest.Platform != target.Platform || manifest.Filename != filename ||
		len(manifest.SHA256) != sha256.Size*2 || !strings.EqualFold(manifest.SHA256, checksum) {
		return errManifestInvalid
	}
	if _, err := hex.DecodeString(manifest.SHA256); err != nil {
		return errors.Join(errManifestInvalid, err)
	}
	return nil
}

func verifySHA256(path string, expectedHex string) error {
	// #nosec G304 -- path is the fixed updater archive created beside the executable.
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expectedHex) {
		return errChecksumMismatch
	}
	return nil
}

// extractBinary writes the `s-ui/sui` entry from the gzip tarball to dest. Header
// paths are never honored for the output location (extraction is pinned to dest),
// preventing path traversal.
func extractBinary(archive string, dest string) error {
	return extractBinaryLimit(archive, dest, maxArtifactBytes)
}

func extractBinaryLimit(archive string, dest string, limit int64) error {
	if limit < 0 {
		return errArchiveMemberTooLarge
	}
	// #nosec G304 -- archive is the fixed updater archive created beside the executable.
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("archive does not contain s-ui/sui")
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.ToSlash(header.Name)
		if name != "s-ui/sui" && filepath.Base(name) != "sui" {
			continue
		}
		if header.Size < 0 || header.Size > limit {
			return errArchiveMemberTooLarge
		}
		// #nosec G302,G304 -- dest is the fixed staging binary and must be executable.
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		written, copyErr := io.Copy(out, io.LimitReader(tr, sizeProbeLimit(limit)))
		if copyErr != nil {
			return errors.Join(copyErr, out.Close(), removeUpdateFile(dest))
		}
		if written > limit {
			return errors.Join(errArchiveMemberTooLarge, out.Close(), removeUpdateFile(dest))
		}
		if err := out.Sync(); err != nil {
			return errors.Join(err, out.Close(), removeUpdateFile(dest))
		}
		if err := out.Close(); err != nil {
			return errors.Join(err, removeUpdateFile(dest))
		}
		return nil
	}
}

func copyFile(src string, dst string) error {
	// #nosec G304 -- src is os.Executable() or its fixed updater staging path.
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	// #nosec G302,G304 -- dst is the fixed rollback binary and must be executable.
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		return errors.Join(err, out.Close(), removeUpdateFile(dst))
	}
	if err := out.Sync(); err != nil {
		return errors.Join(err, out.Close(), removeUpdateFile(dst))
	}
	if err := out.Close(); err != nil {
		return errors.Join(err, removeUpdateFile(dst))
	}
	return nil
}

func removeUpdateFile(path string) error {
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func sizeProbeLimit(limit int64) int64 {
	if limit == int64(^uint64(0)>>1) {
		return limit
	}
	return limit + 1
}

// RestoreBackup restores <execPath>.bak over execPath (rollback, SR-012).
//
// It must use rename, not a content copy: rollback runs inside the very
// process that is executing execPath, and on Linux opening a running binary
// for writing fails with ETXTBSY ("text file busy"). Rename replaces the
// directory entry atomically while the running process keeps its old inode,
// which is the same mechanism swapBinary relies on for the forward swap.
func RestoreBackup(execPath string) error {
	backup := execPath + backupSuffix
	if _, err := os.Stat(backup); err != nil {
		return err
	}
	return os.Rename(backup, execPath)
}

func writePendingMarker(execPath string) error {
	return os.WriteFile(execPath+pendingSuffix, []byte("0"), 0o600)
}

// ClearPendingUpdate confirms a successful boot by removing the transaction's
// marker and backup. Without a marker, an unrelated .bak file is left alone.
func ClearPendingUpdate(execPath string) {
	marker := execPath + pendingSuffix
	if err := os.Remove(marker); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Warning("panel update: could not clear pending marker: ", err)
		}
		return
	}
	if err := removeUpdateFile(execPath + backupSuffix); err != nil {
		logger.Warning("panel update: could not remove confirmed update backup: ", err)
	}
}

// CheckPendingUpdate runs at startup before the marker is cleared: it counts how
// many times a freshly-applied binary has failed to reach a clean boot and, past
// the threshold, restores the backup so a verified-but-unbootable release cannot
// brick the panel (SR-012). Returns true if a rollback was performed.
func CheckPendingUpdate(execPath string) bool {
	marker := execPath + pendingSuffix
	// #nosec G304 -- marker is a fixed suffix of os.Executable().
	raw, err := os.ReadFile(marker)
	if err != nil {
		return false
	}
	attempts := 0
	trimmed := strings.TrimSpace(string(raw))
	if trimmed != "" {
		parsed, parseErr := strconv.Atoi(trimmed)
		if parseErr != nil {
			logger.Warning("panel update: invalid pending marker, resetting boot attempts: ", parseErr)
		} else {
			attempts = parsed
		}
	}
	attempts++
	if attempts >= rollbackAfterAttempts {
		restoreErr := RestoreBackup(execPath)
		if restoreErr == nil {
			_ = os.Remove(marker)
			return true
		}
		// Threshold reached but the backup is unavailable: make the failure
		// operator-visible rather than silently boot-looping (SR-012).
		logger.Error("panel update: new binary failed to boot ", attempts,
			" times and the rollback backup is unavailable: ", restoreErr)
	}
	if err := os.WriteFile(marker, fmt.Appendf(nil, "%d", attempts), 0o600); err != nil {
		logger.Error("panel update: could not update pending marker: ", err)
	}
	return false
}
