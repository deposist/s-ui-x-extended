//go:build !windows

package network

import (
	"errors"
	"syscall"
)

func isAddrNotAvailable(err error) bool {
	return errors.Is(err, syscall.EADDRNOTAVAIL)
}
