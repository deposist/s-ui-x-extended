package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
)

// validRuleSetBytes builds a real binary rule-set, so the tests exercise the
// same parser the core uses at startup rather than a stand-in.
func validRuleSetBytes(t *testing.T, domain string) []byte {
	t.Helper()
	var buf bytes.Buffer
	ruleSet := option.PlainRuleSet{
		Rules: []option.HeadlessRule{
			{
				Type: "default",
				DefaultOptions: option.DefaultHeadlessRule{
					DomainSuffix: []string{domain},
				},
			},
		},
	}
	if err := srs.Write(&buf, ruleSet, 3); err != nil {
		t.Fatalf("srs.Write: %v", err)
	}
	return buf.Bytes()
}

func configureTestRuleSetTLSClient(t *testing.T, servers ...*httptest.Server) {
	t.Helper()
	allowed := make(map[string]struct{}, len(servers))
	for _, server := range servers {
		allowed[server.Listener.Addr().String()] = struct{}{}
	}
	original := ruleSetDirectTransportFactory
	ruleSetDirectTransportFactory = func() *http.Transport {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test TLS server only
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			if _, ok := allowed[address]; !ok {
				return nil, fmt.Errorf("unexpected test target %s", address)
			}
			return (&net.Dialer{}).DialContext(ctx, network, address)
		}
		return transport
	}

	originalValidation := validateDirectRuleSetURL
	validateDirectRuleSetURL = func(context.Context, string) error { return nil }
	t.Cleanup(func() { validateDirectRuleSetURL = originalValidation })
	t.Cleanup(func() { ruleSetDirectTransportFactory = original })
}

func TestVerifyRuleSetBytesRejectsCorruptTrailer(t *testing.T) {
	valid := validRuleSetBytes(t, "checksum.example")
	if len(valid) < 8 {
		t.Fatalf("unexpected short fixture: %d", len(valid))
	}

	corrupt := append([]byte(nil), valid...)
	corrupt[len(corrupt)-1] ^= 0xff
	if err := verifyRuleSetBytes(corrupt); err == nil {
		t.Fatal("a corrupt zlib checksum must be rejected")
	}

	if err := verifyRuleSetBytes(valid[:len(valid)-1]); err == nil {
		t.Fatal("a truncated zlib trailer must be rejected")
	}
}

func TestMaterializeWritesVerifiedRuleSets(t *testing.T) {
	initDoctorTestDB(t)
	payload := validRuleSetBytes(t, "example.com")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(payload); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, server)
	assets, err := svc.Materialize([]RuleSetSource{{Tag: "geosite-ru", URL: server.URL + "/geosite.srs"}})
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	// The whole point is that the core can load this file, so verify it the
	// same way the core would.
	if err := VerifyRuleSetFile(assets[0].Path); err != nil {
		t.Fatalf("materialized file is not loadable: %v", err)
	}
}

// Refresh is the replacement for a remote rule-set's update_interval. It can
// only work if materialize recorded where each file came from, since a local
// rule-set in the config carries no URL.
func TestRefreshUpdatesFilesFromManifest(t *testing.T) {
	initDoctorTestDB(t)
	payload := validRuleSetBytes(t, "first.example")
	var hits int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if _, err := w.Write(payload); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, server)
	assets, err := svc.Materialize([]RuleSetSource{{Tag: "geosite-ru", URL: server.URL + "/geosite.srs"}})
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}

	// Serve different content so a successful refresh is observable on disk.
	payload = validRuleSetBytes(t, "second.example")
	count, err := svc.Refresh()
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 refreshed asset, got %d", count)
	}
	if hits != 2 {
		t.Fatalf("expected the source to be fetched again, got %d requests", hits)
	}
	data, err := os.ReadFile(assets[0].Path)
	if err != nil {
		t.Fatalf("read refreshed file: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("refresh did not update the file contents")
	}
}

// With nothing materialized there is nothing to update, and that is a normal
// state rather than an error the operator should see.
func TestRefreshWithoutManifestIsNoOp(t *testing.T) {
	initDoctorTestDB(t)
	svc := RuleSetAssetService{}
	count, err := svc.Refresh()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 refreshed assets, got %d", count)
	}
}

// The existing files are what keep the core startable, so a failed refresh must
// leave them exactly as they were rather than truncating or deleting them.
func TestRefreshFailureKeepsPreviousFile(t *testing.T) {
	initDoctorTestDB(t)
	good := validRuleSetBytes(t, "keep.example")
	serveGood := true
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveGood {
			if _, err := w.Write(good); err != nil {
				t.Errorf("write response: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, server)
	assets, err := svc.Materialize([]RuleSetSource{{Tag: "geoip-ru", URL: server.URL + "/geoip.srs"}})
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}

	serveGood = false
	if _, err := svc.Refresh(); err == nil {
		t.Fatal("expected refresh to fail")
	}
	if err := VerifyRuleSetFile(assets[0].Path); err != nil {
		t.Fatalf("previous file should still be loadable, got %v", err)
	}
}

// A captive portal or a "file not found" page returns HTTP 200 with HTML. That
// must be rejected at download time, because writing it would produce a file
// that kills the core at startup.
func TestMaterializeRejectsNonRuleSetPayload(t *testing.T) {
	initDoctorTestDB(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("<html><body>404 not found</body></html>")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, server)
	_, err := svc.Materialize([]RuleSetSource{{Tag: "geosite-ru", URL: server.URL}})
	if err == nil {
		t.Fatal("Materialize must reject a payload that is not a binary rule-set")
	}
	if _, statErr := os.Stat(RuleSetAssetPath("geosite-ru")); !os.IsNotExist(statErr) {
		t.Fatal("a rejected download must not be written to disk")
	}
}

// A failed refresh must not destroy the previous good file, otherwise a
// transient network problem would turn into a core that cannot start.
func TestMaterializeKeepsPreviousFileOnFailure(t *testing.T) {
	initDoctorTestDB(t)
	good := validRuleSetBytes(t, "example.com")
	okServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(good); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer okServer.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, okServer)
	if _, err := svc.Materialize([]RuleSetSource{{Tag: "geoip-ru", URL: okServer.URL}}); err != nil {
		t.Fatalf("initial Materialize: %v", err)
	}

	failServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failServer.Close()
	if _, err := svc.Materialize([]RuleSetSource{{Tag: "geoip-ru", URL: failServer.URL}}); err == nil {
		t.Fatal("Materialize must fail on HTTP 500")
	}
	if err := VerifyRuleSetFile(RuleSetAssetPath("geoip-ru")); err != nil {
		t.Fatalf("previous good file must survive a failed refresh: %v", err)
	}
}

// Materialize is all-or-nothing: if any source fails, no file is written, so
// the caller can never reference a half-populated directory.
func TestMaterializeIsAllOrNothing(t *testing.T) {
	initDoctorTestDB(t)
	good := validRuleSetBytes(t, "example.com")
	okServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write(good); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer okServer.Close()
	failServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusNotFound)
	}))
	defer failServer.Close()

	svc := RuleSetAssetService{}
	configureTestRuleSetTLSClient(t, okServer, failServer)
	_, err := svc.Materialize([]RuleSetSource{
		{Tag: "geosite-ru", URL: okServer.URL},
		{Tag: "geoip-ru", URL: failServer.URL},
	})
	if err == nil {
		t.Fatal("Materialize must fail when one source fails")
	}
	if _, statErr := os.Stat(RuleSetAssetPath("geosite-ru")); !os.IsNotExist(statErr) {
		t.Fatal("no file may be written when a sibling download fails")
	}
}

// Tags reach this code from API callers, so a traversal attempt must be
// rejected rather than deciding where the panel writes.
func TestMaterializeRejectsUnsafeTagsAndURLs(t *testing.T) {
	initDoctorTestDB(t)
	svc := RuleSetAssetService{}
	cases := []struct {
		name   string
		source RuleSetSource
	}{
		{"path traversal", RuleSetSource{Tag: "../../etc/passwd", URL: "https://example.com/a.srs"}},
		{"separator in tag", RuleSetSource{Tag: "a/b", URL: "https://example.com/a.srs"}},
		{"empty tag", RuleSetSource{Tag: "", URL: "https://example.com/a.srs"}},
		{"file scheme", RuleSetSource{Tag: "ok", URL: "file:///etc/passwd"}},
		{"no host", RuleSetSource{Tag: "ok", URL: "https://"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Materialize([]RuleSetSource{tc.source}); err == nil {
				t.Fatalf("Materialize must reject %s", tc.name)
			}
		})
	}
}

// Outbound mode cannot work without a running core. The error must say so
// instead of silently falling back to a direct download, which would defeat
// the point of choosing an outbound on a censored server.
func TestMaterializeOutboundModeRequiresRunningCore(t *testing.T) {
	initDoctorTestDB(t)
	settingSvc := SettingService{}
	if err := settingSvc.saveSetting("ruleSetDownloadMode", "outbound"); err != nil {
		t.Fatalf("saveSetting mode: %v", err)
	}
	if err := settingSvc.saveSetting("ruleSetDownloadOutbound", "proxy-out"); err != nil {
		t.Fatalf("saveSetting outbound: %v", err)
	}

	svc := RuleSetAssetService{}
	_, err := svc.Materialize([]RuleSetSource{{Tag: "geosite-ru", URL: "https://example.com/a.srs"}})
	if err == nil {
		t.Fatal("outbound mode without a running core must fail")
	}
	if !strings.Contains(err.Error(), "core is not running") {
		t.Fatalf("expected a core-not-running error, got: %v", err)
	}
}

func TestValidateRuleSetSettingInput(t *testing.T) {
	if err := validateRuleSetSettingInput("ruleSetDownloadMode", "sideways", nil); err == nil {
		t.Fatal("an unknown mode must be rejected")
	}
	if err := validateRuleSetSettingInput("ruleSetDownloadMode", "direct", nil); err != nil {
		t.Fatalf("direct mode must be accepted: %v", err)
	}
	// Outbound mode with an empty companion tag in the same save is a
	// misconfiguration that would only surface later, at download time.
	withEmptyTag := map[string]string{"ruleSetDownloadOutbound": ""}
	if err := validateRuleSetSettingInput("ruleSetDownloadMode", "outbound", withEmptyTag); err == nil {
		t.Fatal("outbound mode without a tag must be rejected")
	}
	withTag := map[string]string{"ruleSetDownloadOutbound": "proxy-out"}
	if err := validateRuleSetSettingInput("ruleSetDownloadMode", "outbound", withTag); err != nil {
		t.Fatalf("outbound mode with a tag must be accepted: %v", err)
	}
}

func TestMaterializeConcurrentCallsPreserveManifestEntries(t *testing.T) {
	initDoctorTestDB(t)
	payload := validRuleSetBytes(t, "concurrent.example")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	configureTestRuleSetTLSClient(t, server)

	svc := RuleSetAssetService{}
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, tag := range []string{"first", "second"} {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			_, err := svc.Materialize([]RuleSetSource{{Tag: tag, URL: server.URL + "/" + tag + ".srs"}})
			errors <- err
		}(tag)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent Materialize: %v", err)
		}
	}
	manifest := readManifest()
	for _, tag := range []string{"first", "second"} {
		if manifest[tag] == "" {
			t.Fatalf("manifest lost %q: %#v", tag, manifest)
		}
	}
}

func TestVerifyRuleSetFileRejectsUnboundedInputs(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		if err := VerifyRuleSetFile(t.TempDir()); err == nil {
			t.Fatal("a directory must not be read as a rule-set")
		}
	})

	t.Run("oversized sparse file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "oversized.srs")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(ruleSetMaxBytes + 1); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if err := VerifyRuleSetFile(path); err == nil || !strings.Contains(err.Error(), "larger") {
			t.Fatalf("expected an oversized-file error, got %v", err)
		}
	})
}
