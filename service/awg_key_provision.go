package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deposist/s-ui-x-extended/logger"
)

// awgEnvFileDefault matches the environment file the systemd installer writes
// (install.sh SECRETBOX_ENV_FILE) so the panel and the installer share one key
// across updates by any path, including the web self-update that only swaps
// the binary without running install.sh.
const awgEnvFileDefault = "/etc/s-ui/secretbox.env"

const awgEnvFileOverride = "SUI_SECRETBOX_ENV_FILE"

var errAWGEnvKeyMissing = errors.New("no usable AWG_KEY_ENC entry in the environment file")

// EnsureAWGEncryptionKey makes the AWG device-key encryption key available to
// the process: from the environment, from the installer's environment file, or
// by generating and persisting a new one next to the existing entries. An
// existing valid key is never rotated — rotating it would render every stored
// device key undecryptable — so every path only fills in a missing value.
func EnsureAWGEncryptionKey() error {
	if encoded := strings.TrimSpace(os.Getenv(awgEncryptionKeyEnv)); encoded != "" {
		if _, err := decodeAWGMasterKey(encoded); err == nil {
			return nil
		}
	}
	path := strings.TrimSpace(os.Getenv(awgEnvFileOverride))
	if path == "" {
		if runtime.GOOS == "windows" {
			// The canonical file lives beside the systemd unit; on Windows the
			// service wrapper must provide the variable or the file override.
			return fmt.Errorf("%s is not set; configure it via the service environment or %s", awgEncryptionKeyEnv, awgEnvFileOverride)
		}
		path = awgEnvFileDefault
	}
	key, err := readAWGEnvKey(path)
	if err == nil {
		_ = os.Setenv(awgEncryptionKeyEnv, key)
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, errAWGEnvKeyMissing) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	key, err = generateAWGEnvKey()
	if err != nil {
		return err
	}
	if err := writeAWGEnvKey(path, key); err != nil {
		return fmt.Errorf("store %s in %s: %w", awgEncryptionKeyEnv, path, err)
	}
	_ = os.Setenv(awgEncryptionKeyEnv, key)
	logger.Info("generated ", awgEncryptionKeyEnv, " in ", path)
	return nil
}

// readAWGEnvKey returns the first valid AWG_KEY_ENC entry, matching the
// installer's first-non-empty-wins read so both consumers agree on one value.
func readAWGEnvKey(path string) (string, error) {
	// #nosec G304 G703 -- path is the installer's fixed env file or the admin-set SUI_SECRETBOX_ENV_FILE.
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read env file: %w", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), awgEncryptionKeyEnv+"=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, err := decodeAWGMasterKey(value); err == nil {
			return value, nil
		}
	}
	return "", errAWGEnvKeyMissing
}

func generateAWGEnvKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", fmt.Errorf("generate AWG key material: %w", err)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// writeAWGEnvKey rewrites the environment file with every existing line except
// AWG_KEY_ENC entries, then appends the new key. Dropping stale (invalid or
// duplicated) entries first keeps first-match reads deterministic for both the
// panel and install.sh. The file is replaced atomically and kept owner-only.
func writeAWGEnvKey(path, key string) error {
	dir := filepath.Dir(path)
	// #nosec G703 -- dir derives from the same fixed/admin-set env file path.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create env dir: %w", err)
	}
	content := []byte(nil)
	// #nosec G304 G703 -- path is the installer's fixed env file or the admin-set SUI_SECRETBOX_ENV_FILE.
	if raw, err := os.ReadFile(path); err == nil {
		var kept []string
		for _, line := range strings.Split(string(raw), "\n") {
			if _, ok := strings.CutPrefix(strings.TrimSpace(line), awgEncryptionKeyEnv+"="); ok {
				continue
			}
			kept = append(kept, line)
		}
		content = []byte(strings.Join(kept, "\n"))
		if len(content) > 0 && !bytesHasTrailingNewline(content) {
			content = append(content, '\n')
		}
	}
	content = append(content, []byte(awgEncryptionKeyEnv+"="+key+"\n")...)
	temp, err := os.CreateTemp(dir, ".secretbox-*")
	if err != nil {
		return fmt.Errorf("create temp env file: %w", err)
	}
	tempName := temp.Name()
	// #nosec G703 -- temp name derives from the fixed/admin-set env file dir.
	removeTemp := func() { _ = os.Remove(tempName) }
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		removeTemp()
		return fmt.Errorf("write temp env file: %w", err)
	}
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		removeTemp()
		return fmt.Errorf("restrict temp env file: %w", err)
	}
	if err := temp.Close(); err != nil {
		removeTemp()
		return fmt.Errorf("close temp env file: %w", err)
	}
	// #nosec G703 -- target is the same fixed/admin-set env file path.
	if err := os.Rename(tempName, path); err != nil {
		removeTemp()
		return fmt.Errorf("replace env file: %w", err)
	}
	return nil
}

func bytesHasTrailingNewline(content []byte) bool {
	return len(content) > 0 && content[len(content)-1] == '\n'
}
