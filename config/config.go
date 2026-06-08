package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

func GetVersion() string {
	return strings.TrimSpace(version)
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := strings.ToLower(strings.TrimSpace(os.Getenv("SUI_LOG_LEVEL")))
	if logLevel == "" {
		return Info
	}
	level := LogLevel(logLevel)
	if isValidLogLevel(level) {
		return level
	}
	fmt.Fprintf(os.Stderr, "WARNING - invalid SUI_LOG_LEVEL %q; falling back to %q\n", logLevel, Info)
	return Info
}

func isValidLogLevel(level LogLevel) bool {
	switch level {
	case Debug, Info, Warn, Error:
		return true
	default:
		return false
	}
}

func IsDebug() bool {
	return os.Getenv("SUI_DEBUG") == "true"
}

func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("SUI_DB_FOLDER")
	if dbFolderPath == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			// Cross-platform fallback path
			if runtime.GOOS == "windows" {
				return "C:\\Program Files\\s-ui\\db"
			}
			return "/usr/local/s-ui/db"
		}
		dbFolderPath = filepath.Join(dir, "db")
	}
	return dbFolderPath
}

func GetDBPath() string {
	return filepath.Join(GetDBFolderPath(), fmt.Sprintf("%s.db", GetName()))
}

func GetSecret() string {
	if secret := os.Getenv("SUI_SECRET"); secret != "" {
		return secret
	}
	return GetName() + ":" + GetDBFolderPath()
}

func GetForceCookieSecureEnv() (bool, bool, error) {
	raw := strings.TrimSpace(os.Getenv("SUI_FORCE_COOKIE_SECURE"))
	if raw == "" {
		return false, false, nil
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return false, true, err
	}
	return enabled, true, nil
}
