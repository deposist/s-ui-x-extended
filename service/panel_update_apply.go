package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

var errChecksumMismatch = errors.New("artifact checksum does not match the published value")

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
func applyPipeline(target ReleaseTarget, deps panelUpdateDeps, setStage func(UpdateStage)) error {
	if deps.execPath == "" {
		return errors.New("cannot locate current executable")
	}
	dir := filepath.Dir(deps.execPath)

	setStage(UpdateStageDownloading)
	archive := filepath.Join(dir, ".sui-update.tar.gz")
	defer os.Remove(archive)
	if err := downloadToFile(deps.client, target.AssetURL, archive); err != nil {
		return err
	}
	expected, err := downloadChecksum(deps.client, target.ChecksumURL)
	if err != nil {
		return err
	}

	setStage(UpdateStageVerifying)
	if err := verifySHA256(archive, expected); err != nil {
		return err
	}

	setStage(UpdateStageApplying)
	return swapBinary(archive, deps.execPath)
}

// swapBinary extracts the new binary next to the current one and atomically
// replaces it, keeping the previous binary as <exec>.bak for rollback.
func swapBinary(archive string, execPath string) error {
	newBin := execPath + ".new"
	if err := extractBinary(archive, newBin); err != nil {
		os.Remove(newBin)
		return err
	}
	if err := os.Chmod(newBin, 0o755); err != nil {
		os.Remove(newBin)
		return err
	}
	if err := copyFile(execPath, execPath+backupSuffix); err != nil {
		os.Remove(newBin)
		return err
	}
	if err := os.Rename(newBin, execPath); err != nil {
		os.Remove(newBin) // live binary is untouched; .bak remains for safety
		return err
	}
	return nil
}

func downloadToFile(client httpDoer, url string, dest string) error {
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
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, maxArtifactBytes)); err != nil {
		return err
	}
	return f.Sync()
}

func downloadChecksum(client httpDoer, url string) (string, error) {
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumBytes))
	if err != nil {
		return "", err
	}
	// Format produced by `sha256sum`: "<hex>  <filename>".
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty checksum file")
	}
	return strings.ToLower(fields[0]), nil
}

func verifySHA256(path string, expectedHex string) error {
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
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, io.LimitReader(tr, maxArtifactBytes)); err != nil {
			return err
		}
		return out.Sync()
	}
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
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

// ClearPendingUpdate removes the pending-update marker. The freshly-booted new
// binary calls this once it has started successfully, so a clean boot does not
// trigger a rollback.
func ClearPendingUpdate(execPath string) {
	_ = os.Remove(execPath + pendingSuffix)
}

// CheckPendingUpdate runs at startup before the marker is cleared: it counts how
// many times a freshly-applied binary has failed to reach a clean boot and, past
// the threshold, restores the backup so a verified-but-unbootable release cannot
// brick the panel (SR-012). Returns true if a rollback was performed.
func CheckPendingUpdate(execPath string) bool {
	marker := execPath + pendingSuffix
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
