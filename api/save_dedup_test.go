package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"github.com/gin-gonic/gin"
)

// authedSaveRouter builds the production router and an authenticated session +
// CSRF token, ready to POST /api/save.
func authedSaveRouter(t *testing.T) (*gin.Engine, string, []*http.Cookie) {
	t.Helper()
	settingService := initSessionTestDB(t)
	router := newSecurityCSRFTestRouter(t, settingService)
	login := performCSRFRequest(router, http.MethodGet, "/login", "")
	if login.Code != http.StatusNoContent {
		t.Fatalf("login returned %d", login.Code)
	}
	token, cookies := issueSecurityCSRFToken(t, router, login.Result().Cookies())
	return router, token, cookies
}

func postSave(router *gin.Engine, token string, cookies []*http.Cookie, data string) *httptest.ResponseRecorder {
	form := url.Values{}
	form.Set("object", "clients")
	form.Set("action", "new")
	form.Set("data", data)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(csrfHeader, token)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	router.ServeHTTP(rec, req)
	return rec
}

func countClients(t *testing.T, name string) int64 {
	t.Helper()
	var count int64
	if err := database.GetDB().Model(&model.Client{}).Where("name = ?", name).Count(&count).Error; err != nil {
		t.Fatalf("count clients: %v", err)
	}
	return count
}

// TestSaveClientCreatesExactlyOneRow is the Phase 0 bisection that exonerated the
// backend: ONE /api/save request inserts exactly one row.
func TestSaveClientCreatesExactlyOneRow(t *testing.T) {
	router, token, cookies := authedSaveRouter(t)
	rec := postSave(router, token, cookies, `{"name":"single","enable":true,"inbounds":[],"links":[]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/save returned %d body=%s", rec.Code, rec.Body.String())
	}
	if got := countClients(t, "single"); got != 1 {
		t.Fatalf("one /api/save produced %d client rows (want 1)", got)
	}
}

// TestSaveDedupBlocksRapidDuplicateCreate proves the authoritative server-side
// guard: the same create submitted twice in quick succession (double-click /
// client double-send / proxy replay) yields exactly one row, not two.
func TestSaveDedupBlocksRapidDuplicateCreate(t *testing.T) {
	router, token, cookies := authedSaveRouter(t)
	payload := `{"name":"dupe","enable":true,"inbounds":[],"links":[]}`
	if rec := postSave(router, token, cookies, payload); rec.Code != http.StatusOK {
		t.Fatalf("first save returned %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := postSave(router, token, cookies, payload); rec.Code != http.StatusOK {
		t.Fatalf("second save returned %d body=%s", rec.Code, rec.Body.String())
	}
	if got := countClients(t, "dupe"); got != 1 {
		t.Fatalf("rapid duplicate create produced %d rows (want 1 — dedup failed)", got)
	}
}

// TestSaveDedupInFlightOutlastsWindow proves the guard covers a SLOW save: while
// the first request is still in-flight (e.g. blocked on a synchronous core
// restart that exceeds the post-completion window), an identical resubmit is
// deduplicated regardless of how much wall-clock has elapsed.
func TestSaveDedupInFlightOutlastsWindow(t *testing.T) {
	c := &saveDedupCache{seen: make(map[string]dedupEntry)}
	const k = "key"
	if !c.claim(k, 0) {
		t.Fatal("first claim should succeed")
	}
	// Far beyond saveDedupWindow but still in-flight (no complete yet): duplicate.
	if c.claim(k, int64(30*time.Second)) {
		t.Fatal("in-flight identical claim must be deduped regardless of elapsed time")
	}
	c.complete(k, int64(30*time.Second))
	// Within the window AFTER completion: still a duplicate.
	if c.claim(k, int64(30*time.Second)+int64(time.Second)) {
		t.Fatal("within post-completion window must be deduped")
	}
	// After the window: a fresh identical create is allowed again.
	if !c.claim(k, int64(30*time.Second)+int64(saveDedupWindow)+int64(time.Second)) {
		t.Fatal("after the post-completion window the claim should succeed")
	}
}

func TestSaveDedupReleaseAllowsImmediateRetry(t *testing.T) {
	c := &saveDedupCache{seen: make(map[string]dedupEntry)}
	const k = "key"
	if !c.claim(k, 0) {
		t.Fatal("claim should succeed")
	}
	c.release(k) // the save failed → must not block a retry
	if !c.claim(k, 1) {
		t.Fatal("release must allow an immediate retry of a failed save")
	}
}

func TestSaveDedupStuckInFlightEvicted(t *testing.T) {
	c := &saveDedupCache{seen: make(map[string]dedupEntry)}
	const k = "key"
	if !c.claim(k, 0) {
		t.Fatal("claim should succeed")
	}
	// Never completed (crash mid-save): within the cap it still dedups...
	if c.claim(k, int64(saveDedupMaxInFlight)/2) {
		t.Fatal("in-flight within the safety cap must dedup")
	}
	// ...but past the cap the stuck entry is evicted so the payload is not wedged.
	if !c.claim(k, int64(saveDedupMaxInFlight)+1) {
		t.Fatal("a stuck in-flight entry past the cap must be evicted")
	}
}

// TestSaveDedupAllowsDistinctCreates proves the guard does not over-deduplicate:
// two DIFFERENT creates in quick succession both succeed.
func TestSaveDedupAllowsDistinctCreates(t *testing.T) {
	router, token, cookies := authedSaveRouter(t)
	if rec := postSave(router, token, cookies, `{"name":"alpha","enable":true,"inbounds":[],"links":[]}`); rec.Code != http.StatusOK {
		t.Fatalf("alpha save returned %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := postSave(router, token, cookies, `{"name":"beta","enable":true,"inbounds":[],"links":[]}`); rec.Code != http.StatusOK {
		t.Fatalf("beta save returned %d body=%s", rec.Code, rec.Body.String())
	}
	if a, b := countClients(t, "alpha"), countClients(t, "beta"); a != 1 || b != 1 {
		t.Fatalf("distinct creates: alpha=%d beta=%d (want 1,1)", a, b)
	}
}
