package core

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	suiLog "github.com/deposist/s-ui-x-extended/logger"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/observable"
	"github.com/sagernet/sing/service/filemanager"
)

func NewFactory(options log.Options) (log.Factory, error) {
	logOptions := options.Options

	if logOptions.Disabled {
		return log.NewNOPFactory(), nil
	}

	var logWriter io.Writer
	var logFilePath string

	switch logOptions.Output {
	case "":
		logWriter = options.DefaultWriter
		if logWriter == nil {
			logWriter = os.Stderr
		}
	case "stderr":
		logWriter = os.Stderr
	case "stdout":
		logWriter = os.Stdout
	default:
		// Defense-in-depth: never let a configured log path escape the panel
		// directory. An unsafe value (absolute path or "..") that reached the
		// stored config (legacy import, manual DB edit) falls back to stderr
		// instead of letting the (often root) core write to an arbitrary file.
		if !config.IsSafeLogOutputPath(logOptions.Output) {
			suiLog.CoreWarning("ignoring unsafe log.output path; writing to stderr instead: ", logOptions.Output)
			logWriter = os.Stderr
		} else {
			logFilePath = logOptions.Output
		}
	}
	logFormatter := log.Formatter{
		BaseTime:         options.BaseTime,
		DisableColors:    logOptions.DisableColor || logFilePath != "",
		DisableTimestamp: !logOptions.Timestamp && logFilePath != "",
		FullTimestamp:    logOptions.Timestamp,
		TimestampFormat:  "-0700 2006-01-02 15:04:05",
	}
	factory := NewDefaultFactory(
		options.Context,
		logFormatter,
		logWriter,
		logFilePath,
	)
	if logOptions.Level != "" {
		logLevel, err := log.ParseLevel(logOptions.Level)
		if err != nil {
			return nil, common.Error("parse log level", err)
		}
		factory.SetLevel(logLevel)
	} else {
		factory.SetLevel(log.LevelTrace)
	}
	return factory, nil
}

var _ log.Factory = (*defaultFactory)(nil)

type defaultFactory struct {
	ctx        context.Context
	formatter  log.Formatter
	writer     io.Writer
	file       *os.File
	filePath   string
	level      log.Level
	subscriber *observable.Subscriber[log.Entry]
	observer   *observable.Observer[log.Entry]
}

func NewDefaultFactory(
	ctx context.Context,
	formatter log.Formatter,
	writer io.Writer,
	filePath string,
) log.ObservableFactory {
	factory := &defaultFactory{
		ctx:        ctx,
		formatter:  formatter,
		writer:     writer,
		filePath:   filePath,
		level:      log.LevelTrace,
		subscriber: observable.NewSubscriber[log.Entry](128),
	}
	factory.observer = observable.NewObserver[log.Entry](factory.subscriber, 64)
	return factory
}

func (f *defaultFactory) Start() error {
	if f.filePath != "" {
		logFile, err := filemanager.OpenFile(f.ctx, f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		f.writer = logFile
		f.file = logFile
	}
	return nil
}

func (f *defaultFactory) Close() error {
	return common.Close(
		common.PtrOrNil(f.file),
		f.subscriber,
	)
}

func (f *defaultFactory) Level() log.Level {
	return f.level
}

func (f *defaultFactory) SetLevel(level log.Level) {
	f.level = level
}

func (f *defaultFactory) Logger() log.ContextLogger {
	return f.NewLogger("")
}

func (f *defaultFactory) NewLogger(tag string) log.ContextLogger {
	return &observableLogger{f, tag}
}

func (f *defaultFactory) Subscribe() (subscription observable.Subscription[log.Entry], done <-chan struct{}, err error) {
	return f.observer.Subscribe()
}

func (f *defaultFactory) UnSubscribe(sub observable.Subscription[log.Entry]) {
	f.observer.UnSubscribe(sub)
}

// AttachPlatformWriter is required by log.Factory. The panel runs the core
// in-process and consumes its logs through Subscribe, so there is no
// mobile-style platform writer to forward to; the call is accepted and ignored.
func (f *defaultFactory) AttachPlatformWriter(log.PlatformWriter) {
}

type observableLogger struct {
	*defaultFactory
	tag string
}

func (l *observableLogger) Log(ctx context.Context, level log.Level, args []any) {
	if strings.HasPrefix(l.tag, "inbound/mtproxy[") {
		level = mtproxyLogLevel(level, args)
	}
	level = log.OverrideLevelFromContext(level, ctx)
	if level > l.level {
		return
	}
	msg := F.ToString(args...)
	// Protocol handlers can include rejected credentials in authentication errors.
	if strings.HasPrefix(l.tag, "inbound/hysteria[") {
		if summary, _, found := strings.Cut(msg, "authentication failed, auth_str="); found {
			msg = summary + "authentication failed"
		}
	} else if strings.HasPrefix(l.tag, "inbound/tuic[") {
		if summary, _, found := strings.Cut(msg, "authentication: unknown user "); found {
			msg = summary + "authentication: unknown user"
		}
	} else if strings.HasPrefix(l.tag, "inbound/vless[") {
		if summary, _, found := strings.Cut(msg, "unknown UUID:"); found {
			msg = summary + "unknown UUID"
		}
	}
	now := time.Now()
	message := l.formatter.Format(ctx, level, l.tag, msg, now)
	messageSimple := msg
	if l.tag != "" {
		messageSimple = l.tag + ": " + messageSimple
	}
	if ctx != nil {
		if id, ok := log.IDFromContext(ctx); ok {
			messageSimple = F.ToString("[", id.ID, " ", log.FormatDuration(time.Since(id.CreatedAt)), "] ", messageSimple)
		}
	}
	switch level {
	case log.LevelInfo:
		suiLog.CoreInfo(l.tag, msg)
	case log.LevelNotice:
		suiLog.CoreInfo(l.tag, msg)
	case log.LevelWarn:
		suiLog.CoreWarning(l.tag, msg)
	case log.LevelPanic:
		suiLog.CoreError(l.tag, msg)
		panic(message)
	case log.LevelFatal:
		suiLog.CoreError(l.tag, msg)
		os.Exit(1)
	case log.LevelError:
		suiLog.CoreError(l.tag, msg)
	default:
		suiLog.CoreDebug(l.tag, msg)
	}
	// Forward the formatted line to the configured writer. os.Stderr is skipped
	// because suiLog's own stream backend already prints there; every other
	// destination (log file, and the buffer tests capture into) is new.
	if (l.filePath != "" || l.writer != os.Stderr) && l.writer != nil {
		_, _ = l.writer.Write([]byte(message))
	}
	l.subscriber.Emit(log.Entry{Level: level, Message: messageSimple})
}

func (l *observableLogger) Trace(args ...any) {
	l.TraceContext(context.Background(), args...)
}

func (l *observableLogger) Debug(args ...any) {
	l.DebugContext(context.Background(), args...)
}

func (l *observableLogger) Info(args ...any) {
	l.InfoContext(context.Background(), args...)
}

func (l *observableLogger) Notice(args ...any) {
	l.NoticeContext(context.Background(), args...)
}

func (l *observableLogger) Warn(args ...any) {
	l.WarnContext(context.Background(), args...)
}

func (l *observableLogger) Error(args ...any) {
	l.ErrorContext(context.Background(), args...)
}

func (l *observableLogger) Fatal(args ...any) {
	l.FatalContext(context.Background(), args...)
}

func (l *observableLogger) Panic(args ...any) {
	l.PanicContext(context.Background(), args...)
}

func (l *observableLogger) TraceContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelTrace, args)
}

func (l *observableLogger) DebugContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelDebug, args)
}

func (l *observableLogger) InfoContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelInfo, args)
}

func (l *observableLogger) NoticeContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelNotice, args)
}

func (l *observableLogger) WarnContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelWarn, args)
}

func (l *observableLogger) ErrorContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelError, args)
}

func (l *observableLogger) FatalContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelFatal, args)
}

func (l *observableLogger) PanicContext(ctx context.Context, args ...any) {
	l.Log(ctx, log.LevelPanic, args)
}
