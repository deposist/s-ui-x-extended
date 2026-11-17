package service

import (
	"os"
	"path/filepath"
	"testing"
)

func cleanupRecoveryFixture(t *testing.T, phase updateTransactionPhase, live string, backup bool) string {
	t.Helper()
	execPath := filepath.Join(t.TempDir(), "sui")
	if err := os.WriteFile(execPath, []byte(live), 0o600); err != nil {
		t.Fatal(err)
	}
	goodPath := execPath + backupSuffix
	if err := os.WriteFile(goodPath, []byte("KNOWN-GOOD"), 0o600); err != nil {
		t.Fatal(err)
	}
	goodHash, err := fileSHA256(goodPath)
	if err != nil {
		t.Fatal(err)
	}
	candidatePath := execPath + ".candidate"
	if err := os.WriteFile(candidatePath, []byte("CANDIDATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidateHash, err := fileSHA256(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(candidatePath); err != nil {
		t.Fatal(err)
	}
	if !backup {
		if err := os.Remove(goodPath); err != nil {
			t.Fatal(err)
		}
	}
	marker := pendingUpdateMarker{Version: updateMarkerVersion, TransactionID: "0123456789abcdef0123456789abcdef", Phase: phase, CandidateSHA256: candidateHash, BackupSHA256: goodHash}
	if err := writePendingUpdateMarker(execPath, marker); err != nil {
		t.Fatal(err)
	}
	return execPath
}

func blockUpdateStagingCleanup(t *testing.T, execPath string) func() {
	t.Helper()
	path := execPath + ".new"
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(path, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Remove(blocker); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecoverPendingUpdateResumesConfirmedCleanup(t *testing.T) {
	execPath := cleanupRecoveryFixture(t, updatePhaseBooting, "CANDIDATE", true)
	unblock := blockUpdateStagingCleanup(t, execPath)
	if err := ConfirmPendingUpdate(execPath); err == nil {
		t.Fatal("blocked cleanup unexpectedly succeeded")
	}
	unblock()
	if rolledBack, err := RecoverPendingUpdate(execPath); err != nil || rolledBack {
		t.Fatalf("confirmed cleanup recovery = (%t, %v)", rolledBack, err)
	}
	if _, err := os.Stat(execPath + pendingSuffix); !os.IsNotExist(err) {
		t.Fatalf("completed marker remains: %v", err)
	}
}

func TestRecoverPendingUpdateResumesPreparedCleanup(t *testing.T) {
	execPath := cleanupRecoveryFixture(t, updatePhasePrepared, "KNOWN-GOOD", true)
	unblock := blockUpdateStagingCleanup(t, execPath)
	if _, err := RecoverPendingUpdate(execPath); err == nil {
		t.Fatal("blocked cleanup unexpectedly succeeded")
	}
	unblock()
	if rolledBack, err := RecoverPendingUpdate(execPath); err != nil || rolledBack {
		t.Fatalf("prepared cleanup recovery = (%t, %v)", rolledBack, err)
	}
}

func TestRecoverPendingUpdateResumesRollbackCleanup(t *testing.T) {
	execPath := cleanupRecoveryFixture(t, updatePhaseBooting, "CANDIDATE", true)
	unblock := blockUpdateStagingCleanup(t, execPath)
	if rolledBack, err := RecoverPendingUpdate(execPath); err == nil || !rolledBack {
		t.Fatalf("blocked rollback cleanup = (%t, %v)", rolledBack, err)
	}
	marker, err := readPendingUpdateMarker(execPath)
	if err != nil || marker.Phase != updatePhaseRolledBack {
		t.Errorf("rollback completion was not persisted: phase=%s err=%v", marker.Phase, err)
	}
	unblock()
	if rolledBack, err := RecoverPendingUpdate(execPath); err != nil || rolledBack {
		t.Fatalf("rollback cleanup recovery = (%t, %v), want no further restart", rolledBack, err)
	}
}

func TestRecoverPendingUpdateRecognizesInterruptedRollbackMarkerWrite(t *testing.T) {
	execPath := cleanupRecoveryFixture(t, updatePhaseBooting, "KNOWN-GOOD", false)
	if rolledBack, err := RecoverPendingUpdate(execPath); err != nil || rolledBack {
		t.Fatalf("post-rename recovery = (%t, %v), want old binary startup", rolledBack, err)
	}
}

func TestRecoverPendingUpdateCleanupRejectsWrongLiveIdentity(t *testing.T) {
	for _, phase := range []updateTransactionPhase{updatePhaseConfirmed, updatePhaseRolledBack, updatePhaseApplied} {
		t.Run(string(phase), func(t *testing.T) {
			execPath := cleanupRecoveryFixture(t, phase, "UNRELATED", false)
			if _, err := RecoverPendingUpdate(execPath); err == nil {
				t.Fatal("unrelated live binary accepted")
			}
			if _, err := os.Stat(execPath + pendingSuffix); err != nil {
				t.Fatalf("unresolved marker removed: %v", err)
			}
		})
	}
}
