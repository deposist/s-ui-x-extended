package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
)

type warpCaptureRoundTripper struct {
	req  *http.Request
	body []byte
}

func (r *warpCaptureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		r.body = body
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
		Header:     http.Header{},
	}, nil
}

func TestSetWarpAuthorizedHeadersIssue31(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "https://api.cloudflareclient.test/v0a4005/reg/device/account", nil)
	if err != nil {
		t.Fatal(err)
	}
	setWarpAuthorizedHeaders(req, "access-token")
	assertWarpClientHeadersIssue31(t, req, "access-token")
	if got := req.Header.Get("Content-Type"); got != "application/json; charset=UTF-8" {
		t.Fatalf("unexpected PUT Content-Type: %q", got)
	}

	customTypeReq, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "https://api.cloudflareclient.test/v0a4005/reg/device", nil)
	if err != nil {
		t.Fatal(err)
	}
	customTypeReq.Header.Set("Content-Type", "application/merge-patch+json")
	setWarpAuthorizedHeaders(customTypeReq, "access-token")
	if got := customTypeReq.Header.Get("Content-Type"); got != "application/merge-patch+json" {
		t.Fatalf("custom Content-Type was overwritten: %q", got)
	}

	getReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.cloudflareclient.test/v0a4005/reg/device", nil)
	if err != nil {
		t.Fatal(err)
	}
	setWarpAuthorizedHeaders(getReq, "")
	if got := getReq.Header.Get("Content-Type"); got != "" {
		t.Fatalf("GET should not receive Content-Type, got %q", got)
	}
	if got := getReq.Header.Get("Authorization"); got != "" {
		t.Fatalf("empty token should not set Authorization, got %q", got)
	}
}

func TestSetWarpLicenseSendsAuthorizedWarpHeadersIssue31(t *testing.T) {
	rt := &warpCaptureRoundTripper{}
	oldClient := warpHTTPClient
	warpHTTPClient = &http.Client{Transport: rt}
	t.Cleanup(func() {
		warpHTTPClient = oldClient
	})

	ep := &model.Endpoint{
		Ext: json.RawMessage(`{
			"access_token": "access-token",
			"device_id": "device-id",
			"license_key": "new-license",
			"api_version": "v0a4005"
		}`),
	}

	if err := (&WarpService{}).SetWarpLicense("old-license", ep); err != nil {
		t.Fatal(err)
	}
	if rt.req == nil {
		t.Fatal("no WARP license request captured")
	}
	if rt.req.Method != http.MethodPut {
		t.Fatalf("unexpected method: %s", rt.req.Method)
	}
	if got := rt.req.URL.Path; got != "/v0a4005/reg/device-id/account" {
		t.Fatalf("unexpected request path: %s", got)
	}
	assertWarpClientHeadersIssue31(t, rt.req, "access-token")
	if got := rt.req.Header.Get("Content-Type"); got != "application/json; charset=UTF-8" {
		t.Fatalf("unexpected Content-Type: %q", got)
	}

	var payload map[string]string
	if err := json.Unmarshal(rt.body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["license"] != "new-license" {
		t.Fatalf("unexpected license payload: %#v", payload)
	}
}

func assertWarpClientHeadersIssue31(t *testing.T, req *http.Request, token string) {
	t.Helper()
	if got := req.Header.Get("User-Agent"); got != warpUserAgent {
		t.Fatalf("unexpected User-Agent: %q", got)
	}
	if got := req.Header.Get("CF-Client-Version"); got != warpClientVersion {
		t.Fatalf("unexpected CF-Client-Version: %q", got)
	}
	if got := req.Header.Get("Accept"); got != "application/json; charset=UTF-8" {
		t.Fatalf("unexpected Accept: %q", got)
	}
	if got := req.Header.Get("Accept-Encoding"); got != "identity" {
		t.Fatalf("unexpected Accept-Encoding: %q", got)
	}
	if token != "" {
		if got := req.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("unexpected Authorization: %q", got)
		}
	}
}
