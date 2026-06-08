package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/logger"
)

func TestGetLogsFilteredValidatesAndFilters(t *testing.T) {
	logger.Init(logger.LevelDebug)
	marker := fmt.Sprintf("f32-%d", time.Now().UnixNano())
	logger.Info(marker, " panel")
	logger.CoreInfo(marker, " core")

	logs, err := (&ServerService{}).GetLogsFiltered("1000", "DEBUG", "core", marker)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one core log, got %#v", logs)
	}
	if !strings.Contains(logs[0], "core") || strings.Contains(logs[0], "panel") {
		t.Fatalf("unexpected filtered log: %#v", logs)
	}

	query, err := ParseLogQuery("1000", "INFO", "panel", "")
	if err != nil {
		t.Fatal(err)
	}
	if query.Count != maxLogCount || query.Level != "info" {
		t.Fatalf("unexpected parsed query: %#v", query)
	}
}

func TestGetLogsFilteredRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name   string
		count  string
		level  string
		source string
		filter string
	}{
		{name: "count", count: "0"},
		{name: "level", level: "trace"},
		{name: "source", source: "kernel"},
		{name: "filter length", filter: strings.Repeat("a", maxLogFilter+1)},
		{name: "filter control", filter: "bad\nfilter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (&ServerService{}).GetLogsFiltered(tt.count, tt.level, tt.source, tt.filter); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
