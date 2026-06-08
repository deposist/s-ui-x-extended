//go:build windows

package network

import (
	"errors"
	"syscall"
)

const windowsEADDRNOTAVAIL = syscall.Errno(10049)

func isAddrNotAvailable(err error) bool {
	return errors.Is(err, syscall.EADDRNOTAVAIL) ||
		errors.Is(err, windowsEADDRNOTAVAIL)
}
