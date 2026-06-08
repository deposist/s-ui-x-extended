package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
)

func TestShowAdminDoesNotPrintPasswordHash(t *testing.T) {
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	t.Cleanup(func() {
		if d := database.GetDB(); d != nil {
			if sqlDB, err := d.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})

	stdout, _ := captureCmdOutput(t, showAdmin)
	if !strings.Contains(stdout, "\tUsername:\t admin") {
		t.Fatalf("showAdmin did not print username: %q", stdout)
	}
	if strings.Contains(stdout, "Password:") {
		t.Fatalf("showAdmin still prints a password field: %q", stdout)
	}
	if strings.Contains(stdout, "bcrypt:") || strings.Contains(stdout, "$2a$") || strings.Contains(stdout, "$2b$") || strings.Contains(stdout, "$2y$") {
		t.Fatalf("showAdmin leaked a password hash: %q", stdout)
	}
	if !strings.Contains(stdout, "Password is hashed; use 's-ui admin -reset' or 's-ui admin -username/-password' to set a new one") {
		t.Fatalf("showAdmin did not print reset guidance: %q", stdout)
	}
}

func captureCmdOutput(t *testing.T, fn func()) (string, string) {
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
