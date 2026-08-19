package cronjob

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
)

type maintenanceMultipartFile struct{ *bytes.Reader }

func (maintenanceMultipartFile) Close() error { return nil }

func TestDatabaseMaintenanceJobDoesNotStartWhileRestoreIsQueued(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SUI_DB_FOLDER", dir)
	path := filepath.Join(dir, "s-ui.db")
	if err := database.InitDB(path); err != nil {
		if strings.Contains(err.Error(), "go-sqlite3 requires cgo") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if live := database.GetDB(); live != nil {
			if sqlDB, err := live.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		_ = os.RemoveAll(dir)
	})
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })

	backup, err := database.GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	activeDone := database.EnterDBOperation()
	restoreDone := make(chan error, 1)
	go func() {
		restoreDone <- database.ImportDB(maintenanceMultipartFile{Reader: bytes.NewReader(backup)}, func() error { return nil })
	}()
	time.Sleep(50 * time.Millisecond) // allow restore to queue its exclusive barrier

	jobRan := make(chan struct{})
	wrapped := databaseMaintenanceJob{Job: cronFuncJob(func() { close(jobRan) })}
	// The tick must skip non-blockingly while the restore drains: it neither
	// starts the job nor occupies the run slot (which is what used to make
	// cron log "cron: skip" at every schedule during a restore).
	wrapped.Run()
	select {
	case <-jobRan:
		t.Fatal("cron DB job ran while restore was waiting to drain")
	case <-time.After(50 * time.Millisecond):
	}

	activeDone()
	if err := <-restoreDone; err != nil {
		t.Fatal(err)
	}
	wrapped.Run()
	select {
	case <-jobRan:
	case <-time.After(time.Second):
		t.Fatal("cron DB job did not run after restore")
	}
}

type cronFuncJob func()

func (f cronFuncJob) Run() { f() }
