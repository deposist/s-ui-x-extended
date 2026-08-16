package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/gin-gonic/gin"
)

type maintenanceBarrierFile struct{ *bytes.Reader }

func (maintenanceBarrierFile) Close() error { return nil }

// AUD-03: API middleware must keep a request's database operation alive until
// its handler returns. Restore must drain that request and prevent a later API
// request from starting DB work while SQLite is being swapped.
// AUD-03: restoration handlers must not retain the shared request lease while
// they call ImportDB for the exclusive restore lease, otherwise they deadlock
// themselves. The rollback endpoint is a distinct ImportDB ingress.
func TestDatabaseOperationMiddlewareAllowsImportXuiRollbackToAcquireRestoreLease(t *testing.T) {
	initSessionTestDB(t)
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })
	t.Cleanup(func() {
		if live := database.GetDB(); live != nil {
			if sqlDB, err := live.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		time.Sleep(50 * time.Millisecond)
	})

	backup, err := database.GetDb("")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(databaseOperationMiddleware())
	router.POST("/api/import-xui/rollback", func(c *gin.Context) {
		if err := database.ImportDB(maintenanceBarrierFile{Reader: bytes.NewReader(backup)}); err != nil {
			t.Errorf("rollback restore failed: %v", err)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/import-xui/rollback", nil))
		done <- recorder
	}()
	select {
	case response := <-done:
		if response.Code != http.StatusNoContent {
			t.Fatalf("rollback restore status=%d", response.Code)
		}
	// A real lease deadlock still trips this; the generous window only absorbs
	// slow first-boot migrations under -race on loaded CI runners, where a
	// single seed insert has been measured near 1s.
	case <-time.After(15 * time.Second):
		t.Fatal("rollback restore deadlocked on its own database request lease")
	}
}

func TestDatabaseOperationMiddlewareDrainsActiveRequestAndBlocksNewRequestDuringRestore(t *testing.T) {
	initSessionTestDB(t)
	database.SetSendSighupHook(func() error { return nil })
	t.Cleanup(func() { database.SetSendSighupHook(nil) })
	// ImportDB installs a new live handle; close that replacement before the
	// temporary DB directory cleanup on Windows releases its file lock.
	t.Cleanup(func() {
		if live := database.GetDB(); live != nil {
			if sqlDB, err := live.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		time.Sleep(50 * time.Millisecond)
	})

	backup, err := database.GetDb("")
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(databaseOperationMiddleware())
	activeStarted := make(chan struct{})
	releaseActive := make(chan struct{})
	router.GET("/active", func(c *gin.Context) {
		close(activeStarted)
		<-releaseActive
		if err := database.GetDB().Exec("SELECT 1").Error; err != nil {
			t.Errorf("active request database operation: %v", err)
		}
		c.Status(http.StatusNoContent)
	})
	router.GET("/new", func(c *gin.Context) {
		if err := database.GetDB().Exec("SELECT 1").Error; err != nil {
			t.Errorf("new request database operation: %v", err)
		}
		c.Status(http.StatusNoContent)
	})

	activeDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/active", nil))
		activeDone <- recorder
	}()
	select {
	case <-activeStarted:
	case <-time.After(time.Second):
		t.Fatal("active request did not enter handler")
	}

	restoreDone := make(chan error, 1)
	go func() {
		restoreDone <- database.ImportDB(maintenanceBarrierFile{Reader: bytes.NewReader(backup)})
	}()
	time.Sleep(50 * time.Millisecond) // allow restore to queue its exclusive barrier

	newDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/new", nil))
		newDone <- recorder
	}()
	select {
	case response := <-newDone:
		t.Fatalf("new API request bypassed restore barrier: status=%d", response.Code)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseActive)
	if response := <-activeDone; response.Code != http.StatusNoContent {
		t.Fatalf("active request status=%d", response.Code)
	}
	if err := <-restoreDone; err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	select {
	case response := <-newDone:
		if response.Code != http.StatusNoContent {
			t.Fatalf("new request status after restore=%d", response.Code)
		}
	case <-time.After(time.Second):
		t.Fatal("new API request did not resume after restore")
	}
}
