package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	if _, err := applyPipeline(target, deps, func(UpdateStage) {}); err != nil {
		t.Fatalf("signed update failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); !bytes.Equal(got, []byte("NEW-BINARY")) {
		t.Fatalf("binary = %q, want signed artifact", got)
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
			_, err := applyPipeline(target, panelUpdateDeps{client: server.Client(), execPath: execPath}, func(UpdateStage) {})
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
			err := downloadToFileLimit(server.Client(), server.URL, dest, limit)
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
	if _, err := downloadChecksumLimit(server.Client(), server.URL, limit); !errors.Is(err, errChecksumTooLarge) {
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
	if _, err := applyPipeline(target, deps, func(UpdateStage) {}); err != errManifestInvalid {
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
	if _, err := applyPipeline(target, deps, func(UpdateStage) {}); err != nil {
		t.Fatalf("apply pipeline failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); !bytes.Equal(got, newContent) {
		t.Fatalf("binary was not replaced with the new content")
	}
	if got, _ := os.ReadFile(execPath + backupSuffix); !bytes.Equal(got, oldContent) {
		t.Fatalf("previous binary was not backed up for rollback")
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
	if err := os.Mkdir(execPath+pendingSuffix, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(execPath + pendingSuffix) })

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
	if _, err := os.Stat(execPath + backupSuffix); !os.IsNotExist(err) {
		t.Fatalf("current transaction backup was not consumed: %v", err)
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

	swapped, err := applyPipeline(target, panelUpdateDeps{client: server.Client(), execPath: execPath}, func(UpdateStage) {})
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

func TestClearPendingUpdateRemovesMarkerAndBackup(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	if err := os.WriteFile(execPath+pendingSuffix, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}

	ClearPendingUpdate(execPath)

	for _, path := range []string{execPath + pendingSuffix, execPath + backupSuffix} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("successful boot left update artifact %q: %v", path, err)
		}
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
