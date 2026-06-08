package database

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInitUserWritesInitialAdminPasswordFileWithoutLeakingSecret(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dbDir)
	dbPath := filepath.Join(dbDir, "s-ui.db")

	var initErr error
	stdout, stderr := captureStdoutStderr(t, func() {
		initErr = InitDB(dbPath)
	})
	if initErr != nil {
		if strings.Contains(initErr.Error(), "go-sqlite3 requires cgo") {
			t.Skip(initErr)
		}
		t.Fatal(initErr)
	}
	t.Cleanup(func() { closeMainDB(t) })

	passwordPath := filepath.Join(dbDir, initialAdminPasswordFile)
	data, err := os.ReadFile(passwordPath)
	if err != nil {
		t.Fatal(err)
	}
	password := strings.TrimSpace(string(data))
	if password == "" {
		t.Fatal("initial admin password file is empty")
	}
	if strings.Contains(stdout, password) {
		t.Fatal("initial admin password leaked to stdout")
	}
	if strings.Contains(stderr, password) {
		t.Fatal("initial admin password leaked to stderr")
	}
	if !strings.Contains(stderr, "initial admin password saved to "+passwordPath+"; delete after first login") {
		t.Fatalf("stderr did not contain the one-time file notice: %q", stderr)
	}
	if count := strings.Count(stderr, "initial admin password saved to "); count != 1 {
		t.Fatalf("expected one stderr notice, got %d: %q", count, stderr)
	}
	info, err := os.Stat(passwordPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("initial admin password file mode = %o, want 0600", info.Mode().Perm())
	}

	closeMainDB(t)
	stdout, stderr = captureStdoutStderr(t, func() {
		initErr = InitDB(dbPath)
	})
	if initErr != nil {
		t.Fatal(initErr)
	}
	if strings.Contains(stdout, password) || strings.Contains(stderr, password) {
		t.Fatal("initial admin password leaked on subsequent startup")
	}
	if strings.Contains(stderr, "initial admin password saved to ") {
		t.Fatalf("subsequent startup repeated the one-time stderr notice: %q", stderr)
	}
	if !strings.Contains(stdout, "initial admin password file still exists") {
		t.Fatalf("subsequent startup did not warn about leftover password file: stdout=%q stderr=%q", stdout, stderr)
	}
}

func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		t.Fatal(err)
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	stdoutDone := make(chan error, 1)
	stderrDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(&stdoutBuf, stdoutR)
		stdoutDone <- err
	}()
	go func() {
		_, err := io.Copy(&stderrBuf, stderrR)
		stderrDone <- err
	}()

	os.Stdout = stdoutW
	os.Stderr = stderrW
	fn()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	_ = stdoutW.Close()
	_ = stderrW.Close()
	if err := <-stdoutDone; err != nil {
		t.Fatal(err)
	}
	if err := <-stderrDone; err != nil {
		t.Fatal(err)
	}
	_ = stdoutR.Close()
	_ = stderrR.Close()

	return stdoutBuf.String(), stderrBuf.String()
}
