package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/deposist/s-ui-x/service"
	"golang.org/x/term"
)

func runDecryptBackup(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, getenv func(string) string) int {
	fs := flag.NewFlagSet("decrypt-backup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var inPath string
	var outPath string
	var passphraseStdin bool
	var passphraseEnv string
	fs.StringVar(&inPath, "in", "", "path to encrypted backup envelope")
	fs.StringVar(&outPath, "out", "", "path to decrypted SQLite database")
	fs.BoolVar(&passphraseStdin, "passphrase-stdin", false, "read backup passphrase from stdin")
	fs.StringVar(&passphraseEnv, "passphrase-env", "", "environment variable containing backup passphrase")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "decrypt-backup: unexpected positional arguments")
		return 2
	}
	if strings.TrimSpace(inPath) == "" || strings.TrimSpace(outPath) == "" {
		fmt.Fprintln(stderr, "decrypt-backup: --in and --out are required")
		return 2
	}
	if passphraseStdin && strings.TrimSpace(passphraseEnv) != "" {
		fmt.Fprintln(stderr, "decrypt-backup: use only one of --passphrase-stdin or --passphrase-env")
		return 2
	}

	passphrase, err := readDecryptBackupPassphrase(stdin, stderr, getenv, passphraseStdin, passphraseEnv)
	if err != nil {
		fmt.Fprintln(stderr, "decrypt-backup:", err)
		return 2
	}
	defer wipeCmdBytes(passphrase)
	if len(passphrase) == 0 {
		fmt.Fprintln(stderr, "decrypt-backup: empty passphrase")
		return 2
	}

	// #nosec G304 -- inPath is a CLI argument supplied by the operator.
	envelope, err := os.ReadFile(inPath)
	if err != nil {
		fmt.Fprintln(stderr, "decrypt-backup:", err)
		return 1
	}
	defer wipeCmdBytes(envelope)
	plaintext, err := service.OpenTelegramBackupEnvelope(envelope, passphrase)
	if err != nil {
		_ = os.Remove(outPath)
		fmt.Fprintln(stderr, "decrypt-backup: decryption_failed")
		return 1
	}
	defer wipeCmdBytes(plaintext)
	if err := writeDecryptBackupOutput(outPath, plaintext); err != nil {
		_ = os.Remove(outPath)
		fmt.Fprintln(stderr, "decrypt-backup:", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "decrypt-backup: wrote", outPath)
	return 0
}

func readDecryptBackupPassphrase(stdin io.Reader, stderr io.Writer, getenv func(string) string, passphraseStdin bool, passphraseEnv string) ([]byte, error) {
	if strings.TrimSpace(passphraseEnv) != "" {
		return []byte(getenv(passphraseEnv)), nil
	}
	if file, ok := stdin.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprint(stderr, "Backup passphrase: ")
		passphrase, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(stderr)
		if err != nil {
			return nil, err
		}
		return passphrase, nil
	}
	if !passphraseStdin {
		fmt.Fprintln(stderr, "decrypt-backup: reading passphrase from stdin")
	}
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	return bytes.TrimRight(raw, "\r\n"), nil
}

func writeDecryptBackupOutput(outPath string, plaintext []byte) error {
	dir := filepath.Dir(outPath)
	base := filepath.Base(outPath)
	temp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(plaintext); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, outPath)
}

func wipeCmdBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}
