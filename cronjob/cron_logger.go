package cronjob

import (
	"fmt"

	"github.com/deposist/s-ui-x/logger"
)

// cronLogger adapts the panel logger to robfig/cron's Logger interface.
//
// cron.Recover calls Error when it recovers a panic from a scheduled job, and
// cron.SkipIfStillRunning calls Info when it skips an overlapping run. Skips are
// surfaced at warning level because an overlapping invocation means a job
// overran its schedule and is worth operator attention.
type cronLogger struct{}

func (cronLogger) Info(msg string, keysAndValues ...interface{}) {
	logger.Warning(cronLogLine(msg, keysAndValues))
}

func (cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	logger.Error(cronLogLine(fmt.Sprintf("%s: %v", msg, err), keysAndValues))
}

// cronLogLine renders a cron log message followed by its logfmt-style key/value
// pairs (e.g. cron's recovered-panic stack).
func cronLogLine(msg string, keysAndValues []interface{}) string {
	line := "cron: " + msg
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		line += fmt.Sprintf(" %v=%v", keysAndValues[i], keysAndValues[i+1])
	}
	return line
}
