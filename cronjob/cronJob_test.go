package cronjob

import (
	"testing"
	"time"
)

func TestCronJobStartRegistersJobsSynchronously(t *testing.T) {
	initCronJobTestDB(t)

	c := NewCronJob()
	if err := c.Start(time.UTC, 30); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Stop)

	entries := c.cron.Entries()
	if len(entries) != 13 {
		t.Fatalf("expected 13 registered cron entries immediately after Start, got %d", len(entries))
	}
}
