package service

import (
	"errors"
	"fmt"
	"os"
)

const panelUpdateLockSuffix = ".update-lock"

type panelUpdateFileLock struct {
	file *os.File
}

func acquirePanelUpdateProcessLock(execPath string) (panelUpdateProcessLock, error) {
	if execPath == "" {
		return nil, errors.New("cannot locate current executable")
	}
	// #nosec G304 -- execPath is the validated running executable path; the
	// sibling lock file intentionally follows it across supported install roots.
	file, err := os.OpenFile(execPath+panelUpdateLockSuffix, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open update process lock: %w", err)
	}
	if err := lockPanelUpdateFile(file); err != nil {
		_ = file.Close()
		if errors.Is(err, errUpdateInProgress) {
			return nil, err
		}
		return nil, fmt.Errorf("acquire update process lock: %w", err)
	}
	if err := file.Truncate(0); err != nil {
		_ = unlockPanelUpdateFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("truncate update process lock: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		_ = unlockPanelUpdateFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("seek update process lock: %w", err)
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = unlockPanelUpdateFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("write update process lock: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = unlockPanelUpdateFile(file)
		_ = file.Close()
		return nil, fmt.Errorf("sync update process lock: %w", err)
	}
	return &panelUpdateFileLock{file: file}, nil
}

func (l *panelUpdateFileLock) release() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	return errors.Join(unlockPanelUpdateFile(file), file.Close())
}
