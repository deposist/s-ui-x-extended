//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package logger

import (
	"log/slog"
	"log/syslog"
	"time"
)

type syslogBackend struct {
	writer *syslog.Writer
}

func newSyslogBackend() (logBackend, error) {
	writer, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "s-ui")
	if err != nil {
		return nil, err
	}
	return &syslogBackend{writer: writer}, nil
}

func (b *syslogBackend) Log(_ time.Time, level slog.Level, message string) {
	switch {
	case level < slog.LevelInfo:
		_ = b.writer.Debug(message)
	case level < slog.LevelWarn:
		_ = b.writer.Info(message)
	case level < slog.LevelError:
		_ = b.writer.Warning(message)
	default:
		_ = b.writer.Err(message)
	}
}
