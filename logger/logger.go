package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	defaultSlog *slog.Logger
	logConfigMu sync.RWMutex
	logConfig   = loggerConfig{minLevel: slog.LevelDebug}
	logBufferMu sync.RWMutex
	logBuffer   = newLogRingBuffer(logBufferCapacity)
)

const logBufferCapacity = 10240

type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

type bufferedLog struct {
	time   string
	level  slog.Level
	source string
	log    string
}

type loggerConfig struct {
	backend  logBackend
	minLevel slog.Level
}

type logBackend interface {
	Log(t time.Time, level slog.Level, message string)
}

type streamBackend struct {
	writer      io.Writer
	includeTime bool
	mu          sync.Mutex
}

func newStreamBackend(writer io.Writer, includeTime bool) *streamBackend {
	return &streamBackend{writer: writer, includeTime: includeTime}
}

func (b *streamBackend) Log(t time.Time, level slog.Level, message string) {
	if t.IsZero() {
		t = time.Now()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.includeTime {
		fmt.Fprintf(b.writer, "%s %s - %s\n", t.Format("2006/01/02 15:04:05"), slogLevelName(level), message)
		return
	}
	fmt.Fprintf(b.writer, "%s - %s\n", slogLevelName(level), message)
}

type logRingBuffer struct {
	items []bufferedLog
	next  int
	full  bool
}

func newLogRingBuffer(capacity int) *logRingBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &logRingBuffer{
		items: make([]bufferedLog, 0, capacity),
	}
}

func Init(level Level) {
	backend := initBackend()
	panelLogger := Slog("panel")

	logConfigMu.Lock()
	logConfig = loggerConfig{
		backend:  backend,
		minLevel: levelToSlog(level),
	}
	defaultSlog = panelLogger
	logConfigMu.Unlock()

	slog.SetDefault(panelLogger)
}

func Default() *slog.Logger {
	logConfigMu.RLock()
	current := defaultSlog
	logConfigMu.RUnlock()
	if current != nil {
		return current
	}
	return Slog("panel")
}

func Debug(args ...interface{}) {
	logWithSource("panel", slog.LevelDebug, fmt.Sprint(args...))
}

func Debugf(format string, args ...interface{}) {
	logWithSource("panel", slog.LevelDebug, fmt.Sprintf(format, args...))
}

func Info(args ...interface{}) {
	logWithSource("panel", slog.LevelInfo, fmt.Sprint(args...))
}

func Infof(format string, args ...interface{}) {
	logWithSource("panel", slog.LevelInfo, fmt.Sprintf(format, args...))
}

func Warning(args ...interface{}) {
	logWithSource("panel", slog.LevelWarn, fmt.Sprint(args...))
}

func Warningf(format string, args ...interface{}) {
	logWithSource("panel", slog.LevelWarn, fmt.Sprintf(format, args...))
}

func Error(args ...interface{}) {
	logWithSource("panel", slog.LevelError, fmt.Sprint(args...))
}

func Errorf(format string, args ...interface{}) {
	logWithSource("panel", slog.LevelError, fmt.Sprintf(format, args...))
}

func CoreDebug(args ...interface{}) {
	logCore("DEBUG", fmt.Sprint(args...))
}

func CoreInfo(args ...interface{}) {
	logCore("INFO", fmt.Sprint(args...))
}

func CoreWarning(args ...interface{}) {
	logCore("WARNING", fmt.Sprint(args...))
}

func CoreError(args ...interface{}) {
	logCore("ERROR", fmt.Sprint(args...))
}

func logCore(level string, message string) {
	logWithSource("core", parseSlogLevel(level), message)
}

func logWithSource(source string, level slog.Level, message string) {
	t := time.Now()
	writeConfiguredLog(t, level, message)
	addToBufferAt(source, level, message, t)
}

func writeConfiguredLog(t time.Time, level slog.Level, message string) {
	backend, minLevel := currentLogConfig()
	if level < minLevel {
		return
	}
	backend.Log(t, level, message)
}

func currentLogConfig() (logBackend, slog.Level) {
	logConfigMu.RLock()
	backend := logConfig.backend
	minLevel := logConfig.minLevel
	logConfigMu.RUnlock()

	if backend == nil {
		backend = newStreamBackend(os.Stdout, false)
	}
	return backend, minLevel
}

func initBackend() logBackend {
	_, inContainer := os.LookupEnv("container")
	if !inContainer {
		if _, statErr := os.Stat("/.dockerenv"); statErr == nil {
			inContainer = true
		}
	}
	if inContainer {
		return newStreamBackend(os.Stderr, true)
	}

	backend, err := newSyslogBackend()
	if err == nil {
		return backend
	}
	fmt.Println("Unable to use syslog: " + err.Error())
	return newStreamBackend(os.Stderr, true)
}

func addToBuffer(source string, level string, newLog string) {
	addToBufferAt(source, parseSlogLevel(level), newLog, time.Now())
}

func addToBufferAt(source string, level slog.Level, newLog string, t time.Time) {
	if t.IsZero() {
		t = time.Now()
	}
	logBufferMu.Lock()
	defer logBufferMu.Unlock()

	logBuffer.append(bufferedLog{
		time:   t.Format("2006/01/02 15:04:05"),
		level:  level,
		source: source,
		log:    newLog,
	})
}

func GetLogs(c int, level string) []string {
	return GetLogsFiltered(c, level, "", "")
}

func GetLogsFiltered(c int, level string, source string, filter string) []string {
	var output []string
	minLevel := parseSlogLevel(level)

	logBufferMu.RLock()
	snapshot := logBuffer.snapshot()
	logBufferMu.RUnlock()

	for i := len(snapshot) - 1; i >= 0 && len(output) < c; i-- {
		entry := snapshot[i]
		if source != "" && entry.source != source {
			continue
		}
		if filter != "" && !strings.Contains(entry.log, filter) {
			continue
		}
		if entry.level >= minLevel {
			output = append(output, fmt.Sprintf("%s %s - %s", entry.time, slogLevelName(entry.level), entry.log))
		}
	}
	return output
}

func parseSlogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warning", "warn":
		return slog.LevelWarn
	case "error", "critical":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func levelToSlog(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelWarning:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (r *logRingBuffer) append(entry bufferedLog) {
	if cap(r.items) == 0 {
		r.items = make([]bufferedLog, 0, 1)
	}
	if len(r.items) < cap(r.items) {
		r.items = append(r.items, entry)
		if len(r.items) == cap(r.items) {
			r.full = true
			r.next = 0
		}
		return
	}
	r.items[r.next] = entry
	r.next = (r.next + 1) % len(r.items)
	r.full = true
}

func (r *logRingBuffer) snapshot() []bufferedLog {
	if len(r.items) == 0 {
		return nil
	}
	out := make([]bufferedLog, 0, len(r.items))
	if !r.full {
		return append(out, r.items...)
	}
	out = append(out, r.items[r.next:]...)
	out = append(out, r.items[:r.next]...)
	return out
}
