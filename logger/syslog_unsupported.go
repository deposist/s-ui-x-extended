//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package logger

import "errors"

func newSyslogBackend() (logBackend, error) {
	return nil, errors.New("syslog is unsupported on this platform")
}
