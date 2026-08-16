package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
)

func TestMain(m *testing.M) {
	PanelUpdateDatabaseSnapshot = func() (string, string, error) { return "", "", nil }
	os.Exit(m.Run())
}

func makeTarGz(t *testing.T, suiContent []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "s-ui/sui", Mode: 0o600, Size: int64(len(suiContent)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(suiContent); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func artifactServer(t *testing.T, tarball []byte, checksumHex string) *httptest.Server {
	t.Helper()
	manifest := updateManifest(t, ReleaseTarget{Version: "9.9.9", Channel: "main", Platform: "amd64"}, tarball)
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(tarball) })
	mux.HandleFunc("/checksum", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumHex + "  s-ui-linux-amd64.tar.gz\n"))
	})
	mux.HandleFunc("/asset.manifest.json", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(manifest) })
	server := httptest.NewTLSServer(mux) // https so the SR-003 TLS-only guard passes
	t.Cleanup(server.Close)
	return server
}

func checksumHex(archive []byte) string {
	sum := sha256.Sum256(archive)
	return hex.EncodeToString(sum[:])
}

func updateManifest(t *testing.T, target ReleaseTarget, archive []byte) []byte {
	t.Helper()
	manifest := []byte(`{"version":"` + target.Version + `","channel":"` + target.Channel + `","platform":"` + target.Platform + `","filename":"s-ui-linux-` + target.Platform + `.tar.gz","sha256":"` + checksumHex(archive) + `"}`)
	return manifest
}

// The manifest binds the target metadata and archive digest before the live
// executable is touched.
func TestApplyPipelineRequiresSignedManifestBindingArtifact(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	oldContent := []byte("OLD-WORKING-BINARY")
	if err := os.WriteFile(execPath, oldContent, 0o600); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("NEW-BINARY"))
	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64"}
	manifest := updateManifest(t, target, tarball)

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(tarball) })
	mux.HandleFunc("/checksum", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumHex(tarball) + "  replacement.tar.gz\n"))
	})
	mux.HandleFunc("/asset.manifest.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
	server := httptest.NewTLSServer(mux)
	defer server.Close()

	target.AssetURL = server.URL + "/asset"
	target.ChecksumURL = server.URL + "/checksum"
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if _, err := applyPipeline(context.Background(), target, deps, func(UpdateStage) {}); err != nil {
		t.Fatalf("signed update failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); !bytes.Equal(got, []byte("NEW-BINARY")) {
		t.Fatalf("binary = %q, want signed artifact", got)
	}
}

// A stable release selected while tracking the beta channel carries a main-channel
// manifest. The selected release type, not the tracking channel, defines the
// manifest binding.
func TestApplyPipelineAcceptsStableReleaseSelectedFromBetaChannel(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("OLD-WORKING-BINARY"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := ReleaseTarget{
		Channel:    "beta",
		Version:    "1.0.8",
		Prerelease: false,
		Platform:   "amd64",
	}
	tarball := makeTarGz(t, []byte("STABLE-BINARY"))
	manifestTarget := target
	manifestTarget.Channel = "main"
	manifest := updateManifest(t, manifestTarget, tarball)

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(tarball) })
	mux.HandleFunc("/checksum", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumHex(tarball) + "  s-ui-linux-amd64.tar.gz\n"))
	})
	mux.HandleFunc("/asset.manifest.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
	server := httptest.NewTLSServer(mux)
	defer server.Close()

	target.AssetURL = server.URL + "/asset"
	target.ChecksumURL = server.URL + "/checksum"
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if _, err := applyPipeline(context.Background(), target, deps, func(UpdateStage) {}); err != nil {
		t.Fatalf("stable graduation update failed: %v", err)
	}
	if got, err := os.ReadFile(execPath); err != nil || !bytes.Equal(got, []byte("STABLE-BINARY")) {
		t.Fatalf("binary = %q, err = %v; want stable artifact", got, err)
	}
}

func TestVerifyUpdateManifestRejectsPrereleaseWithMainManifest(t *testing.T) {
	archive := []byte("BETA-ARCHIVE")
	target := ReleaseTarget{
		Channel:    config.UpdateChannelBeta,
		Version:    "1.0.9-beta1",
		Prerelease: true,
		Platform:   "amd64",
	}
	manifestTarget := target
	manifestTarget.Channel = config.UpdateChannelMain
	err := verifyUpdateManifest(updateManifest(t, manifestTarget, archive), target, checksumHex(archive))
	if !errors.Is(err, errManifestInvalid) {
		t.Fatalf("prerelease with main manifest error = %v, want %v", err, errManifestInvalid)
	}
}

func TestApplyPipelineRejectsTamperedSignedReleaseInputs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ReleaseTarget, *[]byte, *[]byte, *[]byte)
	}{
		{name: "archive", mutate: func(_ *ReleaseTarget, archive *[]byte, _ *[]byte, _ *[]byte) {
			*archive = makeTarGz(t, []byte("TAMPERED"))
		}},
		{name: "checksum", mutate: func(_ *ReleaseTarget, _ *[]byte, checksum *[]byte, _ *[]byte) {
			*checksum = []byte("00" + string((*checksum)[2:]))
		}},
		{name: "manifest", mutate: func(_ *ReleaseTarget, _ *[]byte, _ *[]byte, manifest *[]byte) {
			*manifest = bytes.Replace(*manifest, []byte(`"version":"9.9.9"`), []byte(`"version":"9.9.8"`), 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			execPath := filepath.Join(dir, "sui")
			oldContent := []byte("OLD-WORKING-BINARY")
			if err := os.WriteFile(execPath, oldContent, 0o600); err != nil {
				t.Fatal(err)
			}
			archive := makeTarGz(t, []byte("NEW-BINARY"))
			target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64"}
			manifest := updateManifest(t, target, archive)
			checksum := []byte(checksumHex(archive) + "  s-ui-linux-amd64.tar.gz\n")
			tt.mutate(&target, &archive, &checksum, &manifest)
			mux := http.NewServeMux()
			mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) })
			mux.HandleFunc("/checksum", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(checksum) })
			mux.HandleFunc("/asset.manifest.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
			server := httptest.NewTLSServer(mux)
			defer server.Close()
			target.AssetURL, target.ChecksumURL = server.URL+"/asset", server.URL+"/checksum"
			_, err := applyPipeline(context.Background(), target, panelUpdateDeps{client: server.Client(), execPath: execPath}, func(UpdateStage) {})
			if err == nil {
				t.Fatal("tampered signed release was accepted")
			}
			if got, _ := os.ReadFile(execPath); !bytes.Equal(got, oldContent) {
				t.Fatal("live binary changed for tampered release")
			}
		})
	}
}

func TestDownloadToFileEnforcesExactLimit(t *testing.T) {
	const limit = int64(32)
	tests := []struct {
		name    string
		size    int64
		wantErr error
	}{
		{name: "exact limit", size: limit},
		{name: "one byte over", size: limit + 1, wantErr: errArtifactTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("x"), int(tt.size))
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(body)
			}))
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "artifact")
			err := downloadToFileLimit(context.Background(), server.Client(), server.URL, dest, limit)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("download error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
					t.Fatalf("oversized partial artifact was not removed: %v", statErr)
				}
				return
			}
			got, readErr := os.ReadFile(dest)
			if readErr != nil || !bytes.Equal(got, body) {
				t.Fatalf("exact-limit artifact mismatch: bytes=%d err=%v", len(got), readErr)
			}
		})
	}
}

func TestDownloadChecksumRejectsOversize(t *testing.T) {
	const limit = int64(16)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), int(limit+1)))
	}))
	defer server.Close()
	if _, err := downloadChecksumLimit(context.Background(), server.Client(), server.URL, limit); !errors.Is(err, errChecksumTooLarge) {
		t.Fatalf("checksum error = %v, want %v", err, errChecksumTooLarge)
	}
}

func TestExtractBinaryRejectsOversizedMemberWithoutPartialFile(t *testing.T) {
	const limit = int64(32)
	archive := filepath.Join(t.TempDir(), "artifact.tar.gz")
	if err := os.WriteFile(archive, makeTarGz(t, bytes.Repeat([]byte("x"), int(limit+1))), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "sui.new")
	if err := extractBinaryLimit(archive, dest, limit); !errors.Is(err, errArchiveMemberTooLarge) {
		t.Fatalf("extract error = %v, want %v", err, errArchiveMemberTooLarge)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("oversized archive member created a partial executable: %v", err)
	}
}

// T025 / SR-002 / SR-007: a checksum mismatch aborts the apply before the live
// binary is touched.
func TestApplyPipelineRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	oldContent := []byte("OLD-WORKING-BINARY")
	if err := os.WriteFile(execPath, oldContent, 0o600); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("NEW-BINARY"))
	server := artifactServer(t, tarball, "00deadbeef00") // wrong checksum

	target := ReleaseTarget{Channel: "main", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum", Version: "9.9.9"}
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if _, err := applyPipeline(context.Background(), target, deps, func(UpdateStage) {}); err != errManifestInvalid {
		t.Fatalf("expected errManifestInvalid for checksum not bound by the manifest, got %v", err)
	}
	got, _ := os.ReadFile(execPath)
	if !bytes.Equal(got, oldContent) {
		t.Fatalf("live binary was modified despite checksum mismatch")
	}
	if _, err := os.Stat(execPath + backupSuffix); !os.IsNotExist(err) {
		t.Fatalf("no backup should exist when verification fails")
	}
}

// SR-002 happy path: a matching checksum applies the new binary and keeps the
// previous one as a backup (SR-007/SR-012 enabler).
func TestApplyPipelineReplacesBinaryAndKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	oldContent := []byte("OLD-WORKING-BINARY")
	if err := os.WriteFile(execPath, oldContent, 0o600); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("NEW-FRESH-BINARY")
	tarball := makeTarGz(t, newContent)
	sum := sha256.Sum256(tarball)
	server := artifactServer(t, tarball, hex.EncodeToString(sum[:]))

	target := ReleaseTarget{Channel: "main", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum", Version: "9.9.9"}
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if _, err := applyPipeline(context.Background(), target, deps, func(UpdateStage) {}); err != nil {
		t.Fatalf("apply pipeline failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); !bytes.Equal(got, newContent) {
		t.Fatalf("binary was not replaced with the new content")
	}
	if got, _ := os.ReadFile(execPath + backupSuffix); !bytes.Equal(got, oldContent) {
		t.Fatalf("previous binary was not backed up for rollback")
	}
}

func TestSwapBinaryRenameFailureRemovesStagedFiles(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	archive := filepath.Join(dir, "update.tar.gz")
	if err := os.WriteFile(execPath, []byte("CURRENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, makeTarGz(t, []byte("CANDIDATE")), 0o600); err != nil {
		t.Fatal(err)
	}
	renameErr := errors.New("forced rename failure")
	_, err := swapBinaryWithRename(archive, execPath, func(string, string) error { return renameErr })
	if !errors.Is(err, renameErr) {
		t.Fatalf("swap error = %v, want %v", err, renameErr)
	}
	if got, readErr := os.ReadFile(execPath); readErr != nil || string(got) != "CURRENT" {
		t.Fatalf("live binary changed after failed rename: %q, %v", got, readErr)
	}
	for _, path := range []string{execPath + ".new", execPath + backupSuffix + ".new", execPath + backupSuffix + ".previous"} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("failed rename left transaction artifact %q: %v", path, statErr)
		}
	}
	if _, statErr := os.Stat(execPath + backupSuffix); !os.IsNotExist(statErr) {
		t.Fatalf("failed rename left a new transaction backup: %v", statErr)
	}
}

func TestSwapBinaryRenameFailurePreservesPreexistingBackup(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	archive := filepath.Join(dir, "update.tar.gz")
	if err := os.WriteFile(execPath, []byte("CURRENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, makeTarGz(t, []byte("CANDIDATE")), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := swapBinaryWithRename(archive, execPath, func(string, string) error {
		return errors.New("forced rename failure")
	})
	if err == nil {
		t.Fatal("expected rename failure")
	}
	if got, readErr := os.ReadFile(execPath + backupSuffix); readErr != nil || string(got) != "KNOWN-GOOD" {
		t.Fatalf("failed transaction consumed preexisting backup: %q, %v", got, readErr)
	}
}

// T026 / FR-012: a second apply is rejected while one is active.
func TestApplyRejectsConcurrentUpdate(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	panelUpdateState.Lock()
	panelUpdateState.active = true
	panelUpdateState.Unlock()

	if err := (&PanelUpdateService{}).Apply(ReleaseTarget{Version: "9.9.9"}, "admin"); err != errUpdateInProgress {
		t.Fatalf("expected errUpdateInProgress, got %v", err)
	}
}

func TestApplyRejectsUpdateLockedByAnotherProcess(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := first.release(); err != nil {
			t.Errorf("release process lock: %v", err)
		}
	})

	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps { return panelUpdateDeps{execPath: execPath} }
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })

	if err := (&PanelUpdateService{}).Apply(ReleaseTarget{Version: "9.9.9"}, "admin"); !errors.Is(err, errUpdateInProgress) {
		t.Fatalf("Apply with process lock error = %v, want %v", err, errUpdateInProgress)
	}
	if (&PanelUpdateService{}).InProgress() {
		t.Fatal("rejected process-locked update left in-memory guard active")
	}
}

func TestApplyRejectsUnresolvedUpdateTransaction(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath+pendingSuffix, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps { return panelUpdateDeps{execPath: execPath} }
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })

	err := (&PanelUpdateService{}).Apply(ReleaseTarget{Version: "9.9.9"}, "admin")
	if !errors.Is(err, errUpdateRecoveryRequired) {
		t.Fatalf("Apply with unresolved transaction error = %v, want %v", err, errUpdateRecoveryRequired)
	}
	if (&PanelUpdateService{}).InProgress() {
		t.Fatal("rejected recovery transaction left update guard active")
	}
}

// T047 / SR-012: RestoreBackup rolls the live binary back to its backup.
func TestRestoreBackupRollsBack(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RestoreBackup(execPath); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); string(got) != "GOOD-OLD" {
		t.Fatalf("binary was not rolled back, got %q", got)
	}
	// The restore must be a rename, not a content copy: rollback runs inside
	// the process executing execPath, and writing into a running binary fails
	// with ETXTBSY on Linux. A consumed .bak proves the rename path was taken.
	if _, err := os.Stat(execPath + backupSuffix); !os.IsNotExist(err) {
		t.Fatal("backup should be consumed by the rename-based restore")
	}
}

// SR-012: a freshly-applied binary that keeps failing to boot is rolled back
// once the attempt threshold is reached.
func TestCheckPendingUpdateRollsBackAfterThreshold(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	_ = os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o600)
	_ = os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o600)
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}

	// First failed boot: increments, no rollback yet (threshold is 2).
	if CheckPendingUpdate(execPath) {
		t.Fatal("rolled back too early")
	}
	// Second failed boot: threshold reached -> rollback.
	if !CheckPendingUpdate(execPath) {
		t.Fatal("expected rollback at threshold")
	}
	if got, _ := os.ReadFile(execPath); string(got) != "GOOD-OLD" {
		t.Fatalf("binary not rolled back after threshold, got %q", got)
	}
	if _, err := os.Stat(execPath + pendingSuffix); !os.IsNotExist(err) {
		t.Fatal("pending marker should be cleared after rollback")
	}
}

func TestCheckPendingUpdateRollsBackWhenAttemptCannotBePersisted(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}

	if !checkPendingUpdateWithWriter(execPath, func(string, int) error { return errors.New("disk full") }) {
		t.Fatal("marker persistence failure did not fail closed to rollback")
	}
	if got, err := os.ReadFile(execPath); err != nil || string(got) != "GOOD-OLD" {
		t.Fatalf("marker persistence failure did not restore backup: %q, %v", got, err)
	}
}

func TestRecoverPendingUpdateInvalidMarkerFailsClosed(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+pendingSuffix, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	if rolledBack, err := RecoverPendingUpdate(execPath); err == nil || rolledBack {
		t.Fatalf("invalid marker recovery = (%t, %v), want unresolved error", rolledBack, err)
	}
	if got, err := os.ReadFile(execPath); err != nil || string(got) != "BROKEN-NEW" {
		t.Fatalf("invalid marker changed live binary: %q, %v", got, err)
	}
}

func TestRecoverPendingUpdateMigratesLegacyNumericMarker(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+pendingSuffix, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}

	rolledBack, err := RecoverPendingUpdate(execPath)
	if err != nil || rolledBack {
		t.Fatalf("legacy marker recovery = (%t, %v), want successful candidate boot", rolledBack, err)
	}
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if marker.Version != updateMarkerVersion || marker.Phase != updatePhaseApplied || marker.Attempts != 1 {
		t.Fatalf("legacy marker was not migrated before recovery: %#v", marker)
	}
}

func TestLegacyNumericMarkerCompletesHealthyBootLifecycle(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+pendingSuffix, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}

	if rolledBack, err := RecoverPendingUpdate(execPath); err != nil || rolledBack {
		t.Fatalf("recovery = (%t, %v)", rolledBack, err)
	}
	if err := MarkPendingUpdateBooting(execPath); err != nil {
		t.Fatal(err)
	}
	if err := ConfirmPendingUpdate(execPath); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("healthy legacy update left recovery artifact %q: %v", path, err)
		}
	}
}

func TestPreparedPendingUpdateRollsBackAfterCrashBeforeSwapCompletion(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}

	if CheckPendingUpdate(execPath) {
		t.Fatal("prepared transaction rolled back on its first candidate boot")
	}
	if !CheckPendingUpdate(execPath) {
		t.Fatal("prepared transaction did not roll back after repeated candidate boot")
	}
	if got, err := os.ReadFile(execPath); err != nil || string(got) != "KNOWN-GOOD" {
		t.Fatalf("crash recovery did not restore known-good binary: %q, %v", got, err)
	}
}

func TestPreparedTransactionWithUnswappedBinaryCleansUpWithoutRollback(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	newPath := execPath + ".new"
	if err := os.WriteFile(execPath, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarkerForCandidate(execPath, newPath); err != nil {
		t.Fatal(err)
	}

	if CheckPendingUpdate(execPath) {
		t.Fatal("unswapped prepared transaction reported rollback")
	}
	if got, err := os.ReadFile(execPath); err != nil || string(got) != "KNOWN-GOOD" {
		t.Fatalf("unswapped transaction changed live binary: %q, %v", got, err)
	}
	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unswapped transaction left recovery artifact %q: %v", path, err)
		}
	}
}

func TestPendingMarkerRefusesMismatchedBackup(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("UNRELATED"), 0o600); err != nil {
		t.Fatal(err)
	}

	if CheckPendingUpdate(execPath) {
		t.Fatal("mismatched backup was trusted for rollback")
	}
	if got, err := os.ReadFile(execPath); err != nil || string(got) != "CANDIDATE" {
		t.Fatalf("mismatched backup replaced live binary: %q, %v", got, err)
	}
	if _, err := os.Stat(execPath + pendingSuffix); err != nil {
		t.Fatalf("unresolved identity marker was removed: %v", err)
	}
}

func TestFailedUpdateBeforeSwapDoesNotRestorePreviousBackup(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT-B"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("STALE-A"), 0o600); err != nil {
		t.Fatal(err)
	}

	(&PanelUpdateService{}).fail(errors.New("download failed before swap"), execPath, false)

	if got, _ := os.ReadFile(execPath); string(got) != "CURRENT-B" {
		t.Fatalf("pre-swap failure restored stale backup, got %q", got)
	}
	if got, _ := os.ReadFile(execPath + backupSuffix); string(got) != "STALE-A" {
		t.Fatalf("pre-swap failure changed stale backup, got %q", got)
	}
}

func TestMarkerWriteFailureRestoresCurrentTransactionBackup(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT-B"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("STALE-A"), 0o600); err != nil {
		t.Fatal(err)
	}
	markerBlocker := execPath + pendingSuffix + ".blocker"
	if err := os.Mkdir(markerBlocker, 0o700); err != nil {
		t.Fatal(err)
	}
	oldWriter := panelUpdateMarkerWriter
	panelUpdateMarkerWriter = func(string, string) error { return errors.New("forced marker failure") }
	t.Cleanup(func() {
		panelUpdateMarkerWriter = oldWriter
		_ = os.RemoveAll(markerBlocker)
	})

	tarball := makeTarGz(t, []byte("CANDIDATE-C"))
	server := artifactServer(t, tarball, checksumHex(tarball))
	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps {
		return panelUpdateDeps{client: server.Client(), execPath: execPath}
	}
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })

	oldSink := panelUpdateAuditSink
	auditDone := make(chan struct{})
	panelUpdateAuditSink = func(UpdateJob, string, string) { close(auditDone) }
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })

	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum"}
	if err := (&PanelUpdateService{}).Apply(target, "admin"); err != nil {
		t.Fatalf("apply start failed: %v", err)
	}
	waitForUpdateStage(t, UpdateStageFailed)
	select {
	case <-auditDone:
	case <-time.After(15 * time.Second):
		t.Fatal("failed update did not finish its audit sink")
	}

	if got, _ := os.ReadFile(execPath); string(got) != "CURRENT-B" {
		t.Fatalf("marker failure restored wrong binary, got %q", got)
	}
	if got, readErr := os.ReadFile(execPath + backupSuffix); readErr != nil || string(got) != "STALE-A" {
		t.Fatalf("marker failure did not restore preexisting backup: %q, %v", got, readErr)
	}
}

func TestPostRenameSyncFailureReportsSwapForRollback(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT-B"), 0o600); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("CANDIDATE-C"))
	server := artifactServer(t, tarball, checksumHex(tarball))
	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum"}

	oldSync := panelUpdatePostSwapSync
	panelUpdatePostSwapSync = func(string) error { return errors.New("forced directory sync failure") }
	t.Cleanup(func() { panelUpdatePostSwapSync = oldSync })

	swapped, err := applyPipeline(context.Background(), target, panelUpdateDeps{client: server.Client(), execPath: execPath}, func(UpdateStage) {})
	if err == nil || !swapped {
		t.Fatalf("post-rename failure = swapped %t, err %v; want swapped=true with error", swapped, err)
	}
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(UpdateJob, string, string) {}
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })
	(&PanelUpdateService{}).fail(err, execPath, swapped)
	if got, readErr := os.ReadFile(execPath); readErr != nil || string(got) != "CURRENT-B" {
		t.Fatalf("post-rename failure did not restore current binary: %q, %v", got, readErr)
	}
}

func TestPostSwapFailureRestoresCurrentTransactionBackup(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT-B"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("STALE-A"), 0o600); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("CANDIDATE-C"))
	server := artifactServer(t, tarball, checksumHex(tarball))
	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum"}

	swapped, err := applyPipeline(context.Background(), target, panelUpdateDeps{client: server.Client(), execPath: execPath}, func(UpdateStage) {})
	if err != nil || !swapped {
		t.Fatalf("apply pipeline did not report its binary swap: swapped=%t err=%v", swapped, err)
	}
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(UpdateJob, string, string) {}
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })
	(&PanelUpdateService{}).fail(errors.New("failed after swap"), execPath, swapped)

	if got, _ := os.ReadFile(execPath); string(got) != "CURRENT-B" {
		t.Fatalf("post-swap failure restored wrong backup, got %q", got)
	}
	if _, err := os.Stat(execPath + backupSuffix); !os.IsNotExist(err) {
		t.Fatalf("current transaction backup was not consumed: %v", err)
	}
}

type blockingUpdateHTTPDoer struct {
	started chan struct{}
}

func (d *blockingUpdateHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	select {
	case <-d.started:
	default:
		close(d.started)
	}
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestStopPanelUpdateCancelsAndWaitsForActiveDownload(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CURRENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	doer := &blockingUpdateHTTPDoer{started: make(chan struct{})}
	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps { return panelUpdateDeps{client: doer, execPath: execPath} }
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(UpdateJob, string, string) {}
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })

	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64", AssetURL: "https://example.invalid/asset", ChecksumURL: "https://example.invalid/checksum"}
	if err := (&PanelUpdateService{}).Apply(target, "admin"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-doer.started:
	case <-time.After(5 * time.Second):
		t.Fatal("update download did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := StopPanelUpdate(ctx); err != nil {
		t.Fatalf("StopPanelUpdate: %v", err)
	}
	if (&PanelUpdateService{}).InProgress() {
		t.Fatal("cancelled update still active after StopPanelUpdate returned")
	}
	if _, err := os.Stat(execPath + panelUpdateLockSuffix); err != nil {
		t.Fatalf("process lock file should remain reusable after unlock: %v", err)
	}
}

func TestConfirmPendingUpdateRemovesMarkerAndBackup(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("NEW"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		t.Fatal(err)
	}

	if err := MarkPendingUpdateBooting(execPath); err != nil {
		t.Fatal(err)
	}
	if err := ConfirmPendingUpdate(execPath); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("successful boot left update artifact %q: %v", path, err)
		}
	}
}

func TestPendingRecoveryAndConfirmationRespectProcessLock(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	lock, err := acquirePanelUpdateProcessLock(execPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := lock.release(); err != nil {
			t.Errorf("release process lock: %v", err)
		}
	})

	if _, err := RecoverPendingUpdate(execPath); !errors.Is(err, errUpdateInProgress) {
		t.Fatalf("recovery lock error = %v, want %v", err, errUpdateInProgress)
	}
	if err := ConfirmPendingUpdate(execPath); !errors.Is(err, errUpdateInProgress) {
		t.Fatalf("confirmation lock error = %v, want %v", err, errUpdateInProgress)
	}
	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("locked recovery mutated %q: %v", path, err)
		}
	}
}

func TestRecoverPendingUpdateReportsIdentityMismatch(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("UNRELATED"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverPendingUpdate(execPath); err == nil {
		t.Fatal("identity mismatch was treated as no pending update")
	}
}

func TestPendingMarkerCarriesRandomTransactionAndAppliedPhase(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(marker.TransactionID) != 32 || marker.Phase != updatePhasePrepared {
		t.Fatalf("prepared marker = %#v", marker)
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		t.Fatal(err)
	}
	applied, err := readPendingUpdateMarker(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if applied.TransactionID != marker.TransactionID || applied.Phase != updatePhaseApplied {
		t.Fatalf("applied marker = %#v, prepared = %#v", applied, marker)
	}
	if err := MarkPendingUpdateBooting(execPath); err != nil {
		t.Fatal(err)
	}
	booting, err := readPendingUpdateMarker(execPath)
	if err != nil || booting.TransactionID != marker.TransactionID || booting.Phase != updatePhaseBooting {
		t.Fatalf("booting marker = %#v, err = %v", booting, err)
	}
}

func TestPendingRollbackRestoresDatabaseBeforeBinary(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	dbPath := filepath.Join(dir, "s-ui.db")
	snapshotSource := filepath.Join(dir, "snapshot-source.db")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte("MIGRATED"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotSource, []byte("PRE-UPDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSnapshot, oldRestore := PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore
	PanelUpdateDatabaseSnapshot = func() (string, string, error) { return snapshotSource, dbPath, nil }
	PanelUpdateDatabaseRestore = func(snapshotPath, targetPath string) error {
		contents, err := os.ReadFile(snapshotPath)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, contents, 0o600)
	}
	t.Cleanup(func() {
		PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore = oldSnapshot, oldRestore
	})
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		t.Fatal(err)
	}
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		t.Fatal(err)
	}
	marker.Attempts = rollbackAfterAttempts - 1
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := RecoverPendingUpdate(execPath)
	if err != nil || !rolledBack {
		t.Fatalf("recovery = (%t, %v)", rolledBack, err)
	}
	if got, _ := os.ReadFile(execPath); string(got) != "KNOWN-GOOD" {
		t.Fatalf("binary rollback = %q", got)
	}
	if got, _ := os.ReadFile(dbPath); string(got) != "PRE-UPDATE" {
		t.Fatalf("database rollback = %q", got)
	}
}

func TestPendingRollbackDatabaseFailurePreservesTransaction(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	dbPath := filepath.Join(dir, "s-ui.db")
	snapshotSource := filepath.Join(dir, "snapshot-source.db")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotSource, []byte("PRE-UPDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSnapshot, oldRestore := PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore
	PanelUpdateDatabaseSnapshot = func() (string, string, error) { return snapshotSource, dbPath, nil }
	PanelUpdateDatabaseRestore = func(string, string) error { return errors.New("restore failed") }
	t.Cleanup(func() {
		PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore = oldSnapshot, oldRestore
	})
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		t.Fatal(err)
	}
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil {
		t.Fatal(err)
	}
	marker.Attempts = rollbackAfterAttempts - 1
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := RecoverPendingUpdate(execPath)
	if err == nil || rolledBack {
		t.Fatalf("failed DB recovery = (%t, %v), want unresolved", rolledBack, err)
	}
	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix, pendingDatabaseSnapshotPath(execPath, marker)} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("failed DB recovery removed %q: %v", path, err)
		}
	}
	if got, _ := os.ReadFile(execPath); string(got) != "CANDIDATE" {
		t.Fatalf("binary changed before DB recovery: %q", got)
	}
}

func TestBootingTransactionRollsBackImmediatelyAfterMigrationFailure(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	dbPath := filepath.Join(dir, "s-ui.db")
	snapshotSource := filepath.Join(dir, "snapshot-source.db")
	if err := os.WriteFile(execPath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotSource, []byte("PRE-UPDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldSnapshot, oldRestore := PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore
	PanelUpdateDatabaseSnapshot = func() (string, string, error) { return snapshotSource, dbPath, nil }
	PanelUpdateDatabaseRestore = func(snapshotPath, targetPath string) error {
		contents, err := os.ReadFile(snapshotPath)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, contents, 0o600)
	}
	t.Cleanup(func() { PanelUpdateDatabaseSnapshot, PanelUpdateDatabaseRestore = oldSnapshot, oldRestore })
	if err := writePendingMarker(execPath); err != nil {
		t.Fatal(err)
	}
	if err := markPendingUpdateApplied(execPath); err != nil {
		t.Fatal(err)
	}
	if err := MarkPendingUpdateBooting(execPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbPath, []byte("PARTIALLY-MIGRATED"), 0o600); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := RecoverPendingUpdate(execPath)
	if err != nil || !rolledBack {
		t.Fatalf("booting recovery = (%t, %v), want successful rollback", rolledBack, err)
	}
	if got, _ := os.ReadFile(execPath); string(got) != "KNOWN-GOOD" {
		t.Fatalf("binary rollback = %q", got)
	}
	if got, _ := os.ReadFile(dbPath); string(got) != "PRE-UPDATE" {
		t.Fatalf("database rollback = %q", got)
	}
}

func waitForUpdateStage(t *testing.T, want UpdateStage) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if got := (&PanelUpdateService{}).Status().Stage; got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("update stage = %q, want %q", (&PanelUpdateService{}).Status().Stage, want)
}

// SR-006 / SC-008: a failed apply records its terminal OUTCOME (not just the
// attempt) in the audit log, and releases the guard.
func TestFailRecordsFailedOutcomeAudit(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	panelUpdateState.Lock()
	panelUpdateState.active = true
	panelUpdateState.job = &UpdateJob{ID: "upd-1", Channel: "beta", FromVersion: "1.0.0", ToVersion: "2.0.0", Initiator: "admin", Stage: UpdateStageApplying}
	panelUpdateState.Unlock()

	var gotResult, gotErr string
	var gotJob UpdateJob
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(job UpdateJob, result string, errMsg string) { gotResult, gotErr, gotJob = result, errMsg, job }
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })

	(&PanelUpdateService{}).fail(errors.New("boom"), "", false)

	if gotResult != "failed" {
		t.Fatalf("expected failed outcome audit, got %q", gotResult)
	}
	if gotJob.ToVersion != "2.0.0" || gotJob.Initiator != "admin" || gotErr == "" {
		t.Fatalf("audit outcome missing job/error context: job=%#v err=%q", gotJob, gotErr)
	}
	if (&PanelUpdateService{}).InProgress() {
		t.Fatal("guard should be released after a failed apply")
	}
	if st := (&PanelUpdateService{}).Status(); st.Stage != UpdateStageFailed {
		t.Fatalf("stage = %q, want failed", st.Stage)
	}
}

// SR-006 / SC-008: a successful apply records the "applied" outcome BEFORE the
// process exits, and writes the rollback pending-marker.
func TestApplySuccessRecordsAppliedAuditBeforeExit(t *testing.T) {
	resetPanelUpdateStateForTest()
	t.Cleanup(resetPanelUpdateStateForTest)
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("NEW"))
	sum := sha256.Sum256(tarball)
	server := artifactServer(t, tarball, hex.EncodeToString(sum[:]))

	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps {
		return panelUpdateDeps{client: server.Client(), execPath: execPath}
	}
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })

	var auditResult string
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(job UpdateJob, result string, errMsg string) { auditResult = result }
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })

	done := make(chan struct{})
	oldExit := panelUpdateExit
	panelUpdateExit = func() { close(done) }
	t.Cleanup(func() { panelUpdateExit = oldExit })

	target := ReleaseTarget{Channel: "main", Version: "9.9.9", Platform: "amd64", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum"}
	if err := (&PanelUpdateService{}).Apply(target, "admin"); err != nil {
		t.Fatalf("apply start failed: %v", err)
	}
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("apply did not reach restart")
	}
	if auditResult != "applied" {
		t.Fatalf("expected applied outcome audit, got %q", auditResult)
	}
	if got, _ := os.ReadFile(execPath); string(got) != "NEW" {
		t.Fatal("binary was not replaced before restart")
	}
	if _, err := os.Stat(execPath + pendingSuffix); err != nil {
		t.Fatalf("rollback pending-marker not written: %v", err)
	}
}

// The download client must refuse a redirect that downgrades the transfer to
// plain http even though the initial artifact URL is https (SR-003/SR-004).
func TestDownloadToFileRefusesNonHTTPSRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/artifact" {
			w.Header().Set("Location", "http://127.0.0.1:1/evil")
			w.WriteHeader(http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()

	client := panelUpdateHTTPClient()
	client.Transport = server.Client().Transport

	dest := filepath.Join(t.TempDir(), "artifact")
	err := downloadToFile(context.Background(), client, server.URL+"/artifact", dest)
	if err == nil {
		t.Fatal("redirect to non-https url must fail the download")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("failed download must not leave a partial artifact: %v", statErr)
	}
}

func TestDownloadToFileFollowsHTTPSRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			// Relative Location keeps the https origin of the test server.
			w.Header().Set("Location", "/final")
			w.WriteHeader(http.StatusFound)
		case "/final":
			_, _ = w.Write([]byte("payload"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := panelUpdateHTTPClient()
	client.Transport = server.Client().Transport

	dest := filepath.Join(t.TempDir(), "artifact")
	if err := downloadToFile(context.Background(), client, server.URL+"/redirect", dest); err != nil {
		t.Fatalf("https redirect must be followed: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "payload" {
		t.Fatalf("artifact content = %q, %v", got, err)
	}
}
