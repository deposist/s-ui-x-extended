//go:build windows

package service

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

const panelUpdateLockLength = 1

func lockPanelUpdateFile(file *os.File) error {
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		panelUpdateLockLength,
		0,
		new(windows.Overlapped),
	)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return errUpdateInProgress
	}
	return err
}

func unlockPanelUpdateFile(file *os.File) error {
	return windows.UnlockFileEx(
		windows.Handle(file.Fd()),
		0,
		panelUpdateLockLength,
		0,
		new(windows.Overlapped),
	)
}
