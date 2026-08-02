package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/logger"
)

const updateMarkerVersion = 2

type updateTransactionPhase string

const (
	updatePhasePrepared   updateTransactionPhase = "prepared"
	updatePhaseApplied    updateTransactionPhase = "applied"
	updatePhaseBooting    updateTransactionPhase = "booting"
	updatePhaseConfirmed  updateTransactionPhase = "confirmed"
	updatePhaseRolledBack updateTransactionPhase = "rolled_back"
)

type pendingUpdateMarker struct {
	Version         int                    `json:"version"`
	TransactionID   string                 `json:"transactionId"`
	Phase           updateTransactionPhase `json:"phase"`
	Attempts        int                    `json:"attempts"`
	CandidateSHA256 string                 `json:"candidateSha256"`
	BackupSHA256    string                 `json:"backupSha256"`
	DatabasePath    string                 `json:"databasePath,omitempty"`
	DatabaseSHA256  string                 `json:"databaseSha256,omitempty"`
}

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

// PanelUpdateDatabaseSnapshot is installed by the app layer to avoid a
// service/database import cycle. It returns the exact pre-update SQLite image
// and its source path. The snapshot must already be fsynced and self-contained.
var PanelUpdateDatabaseSnapshot func() (snapshotPath string, databasePath string, err error)

// PanelUpdateDatabaseRestore atomically restores the exact pre-update SQLite
// snapshot before the old executable is restarted. The app layer owns the
// database implementation to keep service free of a database import cycle.
var PanelUpdateDatabaseRestore func(snapshotPath string, databasePath string) error

var panelUpdatePostSwapSync = syncDirectory

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
func applyPipeline(ctx context.Context, target ReleaseTarget, deps panelUpdateDeps, setStage func(UpdateStage)) (bool, error) {
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
	if err := downloadToFile(ctx, deps.client, target.AssetURL, archive); err != nil {
		return false, err
	}
	expected, err := downloadChecksum(ctx, deps.client, target.ChecksumURL)
	if err != nil {
		return false, err
	}
	manifest, err := downloadUpdateManifest(ctx, deps.client, target.AssetURL+".manifest.json")
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
	swapped, err := swapBinary(archive, deps.execPath)
	return swapped, err
}

// swapBinary extracts the new binary next to the current one and atomically
// replaces it, keeping the previous binary as <exec>.bak for rollback.
func swapBinary(archive string, execPath string) (bool, error) {
	return swapBinaryWithRename(archive, execPath, os.Rename)
}

func swapBinaryWithRename(archive string, execPath string, rename func(string, string) error) (bool, error) {
	newBin := execPath + ".new"
	backup := execPath + backupSuffix
	backupStage := backup + ".new"
	backupPrevious := backup + ".previous"
	if err := extractBinary(archive, newBin); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin))
	}
	// #nosec G302 -- the staged program must be executable before the atomic swap.
	if err := os.Chmod(newBin, 0o755); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin))
	}
	if err := copyFile(execPath, backupStage); err != nil {
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupStage))
	}
	hadBackup := false
	if _, err := os.Stat(backup); err == nil {
		hadBackup = true
		if err := replaceFile(backup, backupPrevious); err != nil {
			return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupStage))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupStage))
	}
	if err := replaceFile(backupStage, backup); err != nil {
		if hadBackup {
			err = errors.Join(err, replaceFile(backupPrevious, backup))
		}
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupStage), removeUpdateFile(backupPrevious))
	}
	if err := panelUpdateMarkerWriter(execPath, newBin); err != nil {
		if hadBackup {
			err = errors.Join(err, removeUpdateFile(backup), replaceFile(backupPrevious, backup))
		} else {
			err = errors.Join(err, removeUpdateFile(backup))
		}
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupStage), removeUpdateFile(backupPrevious))
	}
	if err := rename(newBin, execPath); err != nil {
		if hadBackup {
			err = errors.Join(err, removeUpdateFile(backup), replaceFile(backupPrevious, backup))
		} else {
			err = errors.Join(err, removeUpdateFile(backup))
		}
		if marker, markerErr := readPendingUpdateMarker(execPath); markerErr == nil {
			err = errors.Join(err, removePendingDatabaseSnapshot(execPath, marker))
		}
		return false, errors.Join(err, removeUpdateFile(newBin), removeUpdateFile(backupPrevious), removeUpdateFile(execPath+pendingSuffix))
	}
	if err := panelUpdatePostSwapSync(filepath.Dir(execPath)); err != nil {
		return true, err
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		return true, err
	}
	return true, removeUpdateFile(backupPrevious)
}

func replaceFile(source, dest string) error {
	return os.Rename(source, dest)
}

func downloadToFile(ctx context.Context, client httpDoer, url string, dest string) error {
	return downloadToFileLimit(ctx, client, url, dest, maxArtifactBytes)
}

func downloadToFileLimit(ctx context.Context, client httpDoer, url string, dest string, limit int64) error {
	if limit < 0 {
		return errArtifactTooLarge
	}
	if !strings.HasPrefix(url, "https://") { // SR-003/SR-004: TLS-only, template URLs
		return fmt.Errorf("refusing non-https artifact url")
	}
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
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

func downloadChecksum(ctx context.Context, client httpDoer, url string) (string, error) {
	return downloadChecksumLimit(ctx, client, url, maxChecksumBytes)
}

func downloadChecksumLimit(ctx context.Context, client httpDoer, url string, limit int64) (string, error) {
	if limit < 0 {
		return "", errChecksumTooLarge
	}
	if !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("refusing non-https checksum url")
	}
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
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

func downloadUpdateManifest(ctx context.Context, client httpDoer, url string) ([]byte, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("refusing non-https manifest url")
	}
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
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
	if err := os.Rename(backup, execPath); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(execPath))
}
func writePendingMarker(execPath string) error {
	return writePendingMarkerForCandidate(execPath, execPath)
}

func writePendingMarkerForCandidate(execPath, candidatePath string) error {
	candidateSHA256, err := fileSHA256(candidatePath)
	if err != nil {
		return err
	}
	backupSHA256, err := fileSHA256(execPath + backupSuffix)
	if err != nil {
		return err
	}
	transactionID, err := newUpdateTransactionID()
	if err != nil {
		return err
	}
	marker := pendingUpdateMarker{
		Version: updateMarkerVersion, TransactionID: transactionID, Phase: updatePhasePrepared,
		CandidateSHA256: candidateSHA256, BackupSHA256: backupSHA256,
	}
	if PanelUpdateDatabaseSnapshot == nil {
		return errors.New("database snapshot handler is unavailable")
	}
	snapshotPath, databasePath, snapshotErr := PanelUpdateDatabaseSnapshot()
	if snapshotErr != nil {
		return fmt.Errorf("snapshot database before update: %w", snapshotErr)
	}
	if snapshotPath != "" {
		ownedPath := execPath + ".update-db-" + transactionID + ".bak"
		if err := replaceFile(snapshotPath, ownedPath); err != nil {
			return fmt.Errorf("install update database snapshot: %w", err)
		}
		databaseSHA256, err := fileSHA256(ownedPath)
		if err != nil {
			return errors.Join(err, removeUpdateFile(ownedPath))
		}
		marker.DatabasePath = databasePath
		marker.DatabaseSHA256 = databaseSHA256
	}
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		return errors.Join(err, removePendingDatabaseSnapshot(execPath, marker))
	}
	return nil
}

func pendingDatabaseSnapshotPath(execPath string, marker pendingUpdateMarker) string {
	if marker.TransactionID == "" || marker.DatabasePath == "" || marker.DatabaseSHA256 == "" {
		return ""
	}
	return execPath + ".update-db-" + marker.TransactionID + ".bak"
}

func removePendingDatabaseSnapshot(execPath string, marker pendingUpdateMarker) error {
	path := pendingDatabaseSnapshotPath(execPath, marker)
	if path == "" {
		return nil
	}
	return removeUpdateFile(path)
}

func newUpdateTransactionID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate update transaction id: %w", err)
	}
	return hex.EncodeToString(id[:]), nil
}

func writePendingUpdateMarker(execPath string, marker pendingUpdateMarker) error {
	encoded, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return writeAtomicUpdateFile(execPath+pendingSuffix, encoded, 0o600)
}

func markPendingUpdateApplied(execPath string) error {
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		return err
	}
	if marker.Phase != updatePhasePrepared {
		return fmt.Errorf("update transaction %s is in unexpected phase %q", marker.TransactionID, marker.Phase)
	}
	marker.Phase = updatePhaseApplied
	return writePendingUpdateMarker(execPath, marker)
}

// ConfirmPendingUpdate confirms a healthy boot. Recovery and confirmation use
// the same inter-process lock as Apply so cleanup cannot race another updater.
// The marker remains authoritative until every owned artifact is removed.
// MarkPendingUpdateBooting durably records that recovery succeeded and startup
// is about to mutate the database. If Init later fails, the next process rolls
// the database and binary back immediately rather than attempting another boot.
func MarkPendingUpdateBooting(execPath string) error {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return err
	}
	defer func() { _ = lock.release() }()
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if marker.Phase != updatePhaseApplied {
		return fmt.Errorf("update transaction %s cannot boot from phase %q", marker.TransactionID, marker.Phase)
	}
	marker.Phase = updatePhaseBooting
	return writePendingUpdateMarker(execPath, marker)
}

func ConfirmPendingUpdate(execPath string) error {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := lock.release(); releaseErr != nil {
			logger.Warning("panel update: release confirmation lock: ", releaseErr)
		}
	}()
	return confirmPendingUpdateLocked(execPath)
}

func confirmPendingUpdateLocked(execPath string) error {
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read pending update for confirmation: %w", err)
	}
	if marker.Phase != updatePhaseBooting && marker.Phase != updatePhaseConfirmed {
		return fmt.Errorf("update transaction %s did not complete its boot", marker.TransactionID)
	}
	if marker.Phase == updatePhaseBooting {
		liveSHA256, err := fileSHA256(execPath)
		if err != nil {
			return fmt.Errorf("verify confirmed binary: %w", err)
		}
		backupSHA256, err := fileSHA256(execPath + backupSuffix)
		if err != nil {
			return fmt.Errorf("verify confirmed backup: %w", err)
		}
		if !strings.EqualFold(liveSHA256, marker.CandidateSHA256) || !strings.EqualFold(backupSHA256, marker.BackupSHA256) {
			return fmt.Errorf("update transaction %s artifact identity mismatch", marker.TransactionID)
		}
		if snapshotPath := pendingDatabaseSnapshotPath(execPath, marker); snapshotPath != "" {
			databaseSHA256, err := fileSHA256(snapshotPath)
			if err != nil || !strings.EqualFold(databaseSHA256, marker.DatabaseSHA256) {
				return errors.Join(fmt.Errorf("update transaction %s database snapshot identity mismatch", marker.TransactionID), err)
			}
		}
		marker.Phase = updatePhaseConfirmed
		if err := writePendingUpdateMarker(execPath, marker); err != nil {
			return fmt.Errorf("persist confirmed update state: %w", err)
		}
	}
	cleanupErr := errors.Join(
		removeUpdateFile(execPath+backupSuffix),
		cleanupUpdateStagingFiles(execPath),
		removePendingDatabaseSnapshot(execPath, marker),
	)
	if cleanupErr != nil {
		return fmt.Errorf("clean confirmed update artifacts: %w", cleanupErr)
	}
	if err := syncDirectory(filepath.Dir(execPath)); err != nil {
		return fmt.Errorf("sync confirmed update cleanup: %w", err)
	}
	if err := removeUpdateFile(execPath + pendingSuffix); err != nil {
		return fmt.Errorf("remove confirmed update marker: %w", err)
	}
	return syncDirectory(filepath.Dir(execPath))
}

// ClearPendingUpdate is retained as a compatibility wrapper for older callers.
// New callers should use ConfirmPendingUpdate and handle its error.
func ClearPendingUpdate(execPath string) {
	if err := ConfirmPendingUpdate(execPath); err != nil {
		logger.Warning("panel update: could not confirm pending transaction: ", err)
	}
}

// RecoverPendingUpdate checks an update transaction during process startup.
// It reports rollback explicitly and never collapses an unresolved recovery
// failure into the ordinary no-pending state.
func RecoverPendingUpdate(execPath string) (bool, error) {
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		return false, err
	}
	defer func() {
		if releaseErr := lock.release(); releaseErr != nil {
			logger.Warning("panel update: release recovery lock: ", releaseErr)
		}
	}()
	return recoverPendingUpdateLocked(execPath, writeBootAttempts)
}

// CheckPendingUpdate is retained for API compatibility. Unresolved recovery is
// logged, but startup must call RecoverPendingUpdate to handle it fatally.
func CheckPendingUpdate(execPath string) bool {
	rolledBack, err := RecoverPendingUpdate(execPath)
	if err != nil {
		logger.Error("panel update: pending recovery failed: ", err)
	}
	return rolledBack
}
func fileSHA256(path string) (string, error) {
	// #nosec G304 -- updater identity paths are fixed beside os.Executable().
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func readPendingUpdateMarker(execPath string) (pendingUpdateMarker, error) {
	markerPath := execPath + pendingSuffix
	// #nosec G304 -- marker is a fixed suffix of os.Executable().
	raw, err := os.ReadFile(markerPath)
	if err != nil {
		return pendingUpdateMarker{}, err
	}
	var marker pendingUpdateMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return pendingUpdateMarker{}, fmt.Errorf("decode pending marker: %w", err)
	}
	if marker.Version != updateMarkerVersion || len(marker.TransactionID) != 32 ||
		(marker.Phase != updatePhasePrepared && marker.Phase != updatePhaseApplied && marker.Phase != updatePhaseBooting &&
			marker.Phase != updatePhaseConfirmed && marker.Phase != updatePhaseRolledBack) ||
		marker.Attempts < 0 || len(marker.CandidateSHA256) != sha256.Size*2 || len(marker.BackupSHA256) != sha256.Size*2 {
		return pendingUpdateMarker{}, errors.New("invalid pending update marker")
	}
	if _, err := hex.DecodeString(marker.TransactionID); err != nil {
		return pendingUpdateMarker{}, fmt.Errorf("invalid update transaction id: %w", err)
	}
	if _, err := hex.DecodeString(marker.CandidateSHA256); err != nil {
		return pendingUpdateMarker{}, fmt.Errorf("invalid candidate digest: %w", err)
	}
	if _, err := hex.DecodeString(marker.BackupSHA256); err != nil {
		return pendingUpdateMarker{}, fmt.Errorf("invalid backup digest: %w", err)
	}
	if marker.DatabasePath == "" != (marker.DatabaseSHA256 == "") {
		return pendingUpdateMarker{}, errors.New("invalid pending database snapshot metadata")
	}
	if marker.DatabaseSHA256 != "" {
		if len(marker.DatabaseSHA256) != sha256.Size*2 {
			return pendingUpdateMarker{}, errors.New("invalid database snapshot digest")
		}
		if _, err := hex.DecodeString(marker.DatabaseSHA256); err != nil {
			return pendingUpdateMarker{}, fmt.Errorf("invalid database snapshot digest: %w", err)
		}
	}
	if marker.DatabasePath != "" && PanelUpdateDatabaseRestore == nil {
		return pendingUpdateMarker{}, errors.New("database rollback handler is unavailable")
	}
	return marker, nil
}

func migrateLegacyPendingUpdateMarker(execPath string) (pendingUpdateMarker, bool, error) {
	// #nosec G304 -- recovery paths are fixed beside os.Executable().
	raw, err := os.ReadFile(execPath + pendingSuffix)
	if err != nil {
		return pendingUpdateMarker{}, false, err
	}
	attempts, ok := parseLegacyBootAttempts(raw)
	if !ok {
		return pendingUpdateMarker{}, false, nil
	}
	candidateSHA256, err := fileSHA256(execPath)
	if err != nil {
		return pendingUpdateMarker{}, true, fmt.Errorf("hash legacy candidate: %w", err)
	}
	backupSHA256, err := fileSHA256(execPath + backupSuffix)
	if err != nil {
		return pendingUpdateMarker{}, true, fmt.Errorf("hash legacy backup: %w", err)
	}
	transactionID, err := newUpdateTransactionID()
	if err != nil {
		return pendingUpdateMarker{}, true, err
	}
	marker := pendingUpdateMarker{
		Version:         updateMarkerVersion,
		TransactionID:   transactionID,
		Phase:           updatePhaseApplied,
		Attempts:        attempts,
		CandidateSHA256: candidateSHA256,
		BackupSHA256:    backupSHA256,
	}
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		return pendingUpdateMarker{}, true, fmt.Errorf("migrate legacy pending marker: %w", err)
	}
	return marker, true, nil
}

func parseLegacyBootAttempts(raw []byte) (int, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return 0, false
	}
	attempts := 0
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return 0, false
		}
		attempts = attempts*10 + int(r-'0')
		if attempts >= rollbackAfterAttempts {
			return 0, false
		}
	}
	return attempts, true
}

func recoverPendingUpdateLocked(execPath string, writeAttempts func(string, int) error) (bool, error) {
	marker, err := readPendingUpdateMarker(execPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		decodeErr := err
		migratedMarker, migrated, migrationErr := migrateLegacyPendingUpdateMarker(execPath)
		if migrationErr != nil {
			return false, migrationErr
		}
		if !migrated {
			return false, decodeErr
		}
		marker = migratedMarker
	}
	backupSHA256, err := fileSHA256(execPath + backupSuffix)
	if err != nil {
		return false, fmt.Errorf("verify transaction %s backup: %w", marker.TransactionID, err)
	}
	if !strings.EqualFold(backupSHA256, marker.BackupSHA256) {
		return false, fmt.Errorf("update transaction %s backup identity mismatch", marker.TransactionID)
	}
	liveSHA256, err := fileSHA256(execPath)
	if err != nil {
		return false, fmt.Errorf("verify transaction %s live binary: %w", marker.TransactionID, err)
	}
	if strings.EqualFold(liveSHA256, marker.BackupSHA256) {
		if marker.Phase != updatePhasePrepared {
			return false, fmt.Errorf("applied transaction %s unexpectedly runs its backup binary", marker.TransactionID)
		}
		if err := cleanupPreparedUpdateLocked(execPath, marker); err != nil {
			return false, err
		}
		return false, nil
	}
	if !strings.EqualFold(liveSHA256, marker.CandidateSHA256) {
		return false, fmt.Errorf("update transaction %s candidate identity mismatch", marker.TransactionID)
	}
	if marker.Phase == updatePhasePrepared {
		// The rename and directory sync completed but the APPLIED marker write was
		// interrupted. Promote the only identity-consistent crash state.
		marker.Phase = updatePhaseApplied
		if err := writePendingUpdateMarker(execPath, marker); err != nil {
			return rollbackPendingUpdateLocked(execPath, marker, fmt.Errorf("persist applied phase: %w", err))
		}
	}
	if marker.Phase == updatePhaseBooting {
		return rollbackPendingUpdateLocked(execPath, marker, nil)
	}
	marker.Attempts++
	if marker.Attempts >= rollbackAfterAttempts {
		return rollbackPendingUpdateLocked(execPath, marker, nil)
	}
	if err := writeAttempts(execPath, marker.Attempts); err != nil {
		return rollbackPendingUpdateLocked(execPath, marker, fmt.Errorf("persist boot attempt: %w", err))
	}
	return false, nil
}

func rollbackCurrentUpdate(execPath string) error {
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		return err
	}
	rolledBack, err := rollbackPendingUpdateLocked(execPath, marker, nil)
	if err != nil {
		return err
	}
	if !rolledBack {
		return errors.New("update transaction was not rolled back")
	}
	return nil
}

func rollbackPendingUpdateLocked(execPath string, marker pendingUpdateMarker, cause error) (bool, error) {
	snapshotPath := pendingDatabaseSnapshotPath(execPath, marker)
	if snapshotPath != "" {
		snapshotSHA256, err := fileSHA256(snapshotPath)
		if err != nil || !strings.EqualFold(snapshotSHA256, marker.DatabaseSHA256) {
			return false, errors.Join(cause, fmt.Errorf("verify transaction %s database snapshot: %w", marker.TransactionID, err))
		}
		if err := PanelUpdateDatabaseRestore(snapshotPath, marker.DatabasePath); err != nil {
			return false, errors.Join(cause, fmt.Errorf("rollback transaction %s database: %w", marker.TransactionID, err))
		}
	}
	if err := RestoreBackup(execPath); err != nil {
		return false, errors.Join(cause, fmt.Errorf("rollback transaction %s binary: %w", marker.TransactionID, err))
	}
	if err := cleanupUpdateStagingFiles(execPath); err != nil {
		return true, errors.Join(cause, err)
	}
	if err := removePendingDatabaseSnapshot(execPath, marker); err != nil {
		return true, errors.Join(cause, fmt.Errorf("remove rolled-back database snapshot: %w", err))
	}
	if err := removeUpdateFile(execPath + pendingSuffix); err != nil {
		return true, errors.Join(cause, fmt.Errorf("remove rolled-back marker: %w", err))
	}
	if err := syncDirectory(filepath.Dir(execPath)); err != nil {
		return true, errors.Join(cause, fmt.Errorf("sync rolled-back transaction: %w", err))
	}
	return true, nil
}

func cleanupPreparedUpdateLocked(execPath string, marker pendingUpdateMarker) error {
	if err := removeUpdateFile(execPath + backupSuffix); err != nil {
		return fmt.Errorf("remove prepared backup: %w", err)
	}
	if err := cleanupUpdateStagingFiles(execPath); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(execPath)); err != nil {
		return fmt.Errorf("sync prepared cleanup: %w", err)
	}
	if err := removePendingDatabaseSnapshot(execPath, marker); err != nil {
		return fmt.Errorf("remove prepared database snapshot: %w", err)
	}
	if err := removeUpdateFile(execPath + pendingSuffix); err != nil {
		return fmt.Errorf("remove prepared marker: %w", err)
	}
	return syncDirectory(filepath.Dir(execPath))
}

func cleanupUpdateStagingFiles(execPath string) error {
	return errors.Join(
		removeUpdateFile(execPath+".new"),
		removeUpdateFile(execPath+backupSuffix+".new"),
		removeUpdateFile(execPath+backupSuffix+".previous"),
		removeUpdateFile(filepath.Join(filepath.Dir(execPath), ".sui-update.tar.gz")),
	)
}

func checkPendingUpdateWithWriter(execPath string, writeAttempts func(string, int) error) bool {
	rolledBack, err := recoverPendingUpdateLocked(execPath, writeAttempts)
	if err != nil {
		logger.Error("panel update: pending recovery failed: ", err)
	}
	return rolledBack
}

func writeBootAttempts(execPath string, attempts int) error {
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		return err
	}
	marker.Attempts = attempts
	return writePendingUpdateMarker(execPath, marker)
}

func writeAtomicUpdateFile(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = removeUpdateFile(tmpPath) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
