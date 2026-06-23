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
	hdr := &tar.Header{Name: "s-ui/sui", Mode: 0o755, Size: int64(len(suiContent)), Typeflag: tar.TypeReg}
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
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(tarball) })
	mux.HandleFunc("/checksum", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksumHex + "  s-ui-linux-amd64.tar.gz\n"))
	})
	server := httptest.NewTLSServer(mux) // https so the SR-003 TLS-only guard passes
	t.Cleanup(server.Close)
	return server
}

// T025 / SR-002 / SR-007: a checksum mismatch aborts the apply before the live
// binary is touched.
func TestApplyPipelineRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	oldContent := []byte("OLD-WORKING-BINARY")
	if err := os.WriteFile(execPath, oldContent, 0o755); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("NEW-BINARY"))
	server := artifactServer(t, tarball, "00deadbeef00") // wrong checksum

	target := ReleaseTarget{AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum", Version: "9.9.9"}
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if err := applyPipeline(target, deps, func(UpdateStage) {}); err != errChecksumMismatch {
		t.Fatalf("expected errChecksumMismatch, got %v", err)
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
	if err := os.WriteFile(execPath, oldContent, 0o755); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("NEW-FRESH-BINARY")
	tarball := makeTarGz(t, newContent)
	sum := sha256.Sum256(tarball)
	server := artifactServer(t, tarball, hex.EncodeToString(sum[:]))

	target := ReleaseTarget{AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum", Version: "9.9.9"}
	deps := panelUpdateDeps{client: server.Client(), execPath: execPath}
	if err := applyPipeline(target, deps, func(UpdateStage) {}); err != nil {
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
	if err := os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RestoreBackup(execPath); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if got, _ := os.ReadFile(execPath); string(got) != "GOOD-OLD" {
		t.Fatalf("binary was not rolled back, got %q", got)
	}
}

// SR-012: a freshly-applied binary that keeps failing to boot is rolled back
// once the attempt threshold is reached.
func TestCheckPendingUpdateRollsBackAfterThreshold(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "sui")
	_ = os.WriteFile(execPath, []byte("BROKEN-NEW"), 0o755)
	_ = os.WriteFile(execPath+backupSuffix, []byte("GOOD-OLD"), 0o755)
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

	(&PanelUpdateService{}).fail(errors.New("boom"), "")

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
	if err := os.WriteFile(execPath, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	tarball := makeTarGz(t, []byte("NEW"))
	sum := sha256.Sum256(tarball)
	server := artifactServer(t, tarball, hex.EncodeToString(sum[:]))

	oldDeps := newPanelUpdateDeps
	newPanelUpdateDeps = func() panelUpdateDeps { return panelUpdateDeps{client: server.Client(), execPath: execPath} }
	t.Cleanup(func() { newPanelUpdateDeps = oldDeps })

	var auditResult string
	oldSink := panelUpdateAuditSink
	panelUpdateAuditSink = func(job UpdateJob, result string, errMsg string) { auditResult = result }
	t.Cleanup(func() { panelUpdateAuditSink = oldSink })

	done := make(chan struct{})
	oldExit := panelUpdateExit
	panelUpdateExit = func() { close(done) }
	t.Cleanup(func() { panelUpdateExit = oldExit })

	target := ReleaseTarget{Channel: "main", Version: "9.9.9", AssetURL: server.URL + "/asset", ChecksumURL: server.URL + "/checksum"}
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
