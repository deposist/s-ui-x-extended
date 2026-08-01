//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package service

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockPanelUpdateFile(file *os.File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return errUpdateInProgress
		}
		return err
	}
	return nil
}

func unlockPanelUpdateFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
