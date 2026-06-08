package logger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Slog returns a standard-library logger backed by the existing logger
// package facade and in-memory log buffer.
func Slog(source string) *slog.Logger {
	if source == "" {
		source = "panel"
	}
	return slog.New(&slogHandler{source: source})
}

type slogHandler struct {
	source string
	attrs  []slog.Attr
	groups []string
}

func (h *slogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *slogHandler) Handle(_ context.Context, record slog.Record) error {
	message := record.Message
	if fields := h.fields(record); fields != "" {
		message += " " + fields
	}
	t := record.Time
	if t.IsZero() {
		t = time.Now()
	}
	writeConfiguredLog(t, record.Level, message)
	addToBufferAt(h.source, record.Level, message, t)
	return nil
}

func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *slogHandler) WithGroup(name string) slog.Handler {
	next := *h
	if name != "" {
		next.groups = append(append([]string{}, h.groups...), name)
	}
	return &next
}

func (h *slogHandler) fields(record slog.Record) string {
	attrs := append([]slog.Attr{}, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	if len(attrs) == 0 {
		return ""
	}
	fields := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		attr.Value = attr.Value.Resolve()
		key := attr.Key
		if len(h.groups) > 0 {
			key = strings.Join(append(append([]string{}, h.groups...), key), ".")
		}
		fields = append(fields, fmt.Sprintf("%s=%s", key, attr.Value.String()))
	}
	return strings.Join(fields, " ")
}

func slogLevelName(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARNING"
	default:
		return "ERROR"
	}
}
