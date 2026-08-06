package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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

// TestParseWarpPeerEndpointAcceptsCloudflareShapes covers the endpoint forms
// Cloudflare actually returns. The `host` value is a DOMAIN, and the `v4`/`v6`
// forms carry a `:0` placeholder port - an IP-only check on `host` rejected
// every real registration with "invalid warp peer endpoint".
func TestParseWarpPeerEndpointAcceptsCloudflareShapes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint map[string]interface{}
		wantAddr string
		wantPort int
	}{
		{
			name: "host is a domain with a real port",
			endpoint: map[string]interface{}{
				"host": "engage.cloudflareclient.com:2408",
			},
			wantAddr: "engage.cloudflareclient.com",
			wantPort: 2408,
		},
		{
			name: "v4 preferred over host",
			endpoint: map[string]interface{}{
				"v4":    "162.159.192.1:0",
				"v6":    "[2606:4700:d0::a29f:c001]:0",
				"host":  "engage.cloudflareclient.com:2408",
				"ports": []interface{}{float64(2408), float64(500)},
			},
			wantAddr: "162.159.192.1",
			wantPort: 2408,
		},
		{
			name: "placeholder port falls back to ports list",
			endpoint: map[string]interface{}{
				"v4":    "162.159.192.1:0",
				"ports": []interface{}{float64(500), float64(2408)},
			},
			wantAddr: "162.159.192.1",
			wantPort: 500,
		},
		{
			name: "placeholder port with no ports list uses 2408",
			endpoint: map[string]interface{}{
				"v4": "162.159.192.1:0",
			},
			wantAddr: "162.159.192.1",
			wantPort: warpDefaultPeerPort,
		},
		{
			name: "bracketed IPv6 is unwrapped",
			endpoint: map[string]interface{}{
				"v6": "[2606:4700:d0::a29f:c001]:2408",
			},
			wantAddr: "2606:4700:d0::a29f:c001",
			wantPort: 2408,
		},
		{
			name: "address without a port",
			endpoint: map[string]interface{}{
				"host": "engage.cloudflareclient.com",
			},
			wantAddr: "engage.cloudflareclient.com",
			wantPort: warpDefaultPeerPort,
		},
		{
			name: "junk v4 falls through to host",
			endpoint: map[string]interface{}{
				"v4":   "not an address:0",
				"host": "engage.cloudflareclient.com:2408",
			},
			wantAddr: "engage.cloudflareclient.com",
			wantPort: 2408,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addr, port, err := parseWarpPeerEndpoint(tc.endpoint)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if addr != tc.wantAddr || port != tc.wantPort {
				t.Fatalf("got %s:%d, want %s:%d", addr, port, tc.wantAddr, tc.wantPort)
			}
		})
	}
}

func TestParseWarpPeerEndpointRejectsUnusable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint map[string]interface{}
	}{
		{"empty", map[string]interface{}{}},
		{"blank host", map[string]interface{}{"host": ""}},
		{"non-string host", map[string]interface{}{"host": float64(2408)}},
		{"garbage host", map[string]interface{}{"host": "!!!:2408"}},
		{"underscore label", map[string]interface{}{"host": "bad_host.example.com:2408"}},
		{"leading dash label", map[string]interface{}{"host": "-bad.example.com:2408"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if addr, port, err := parseWarpPeerEndpoint(tc.endpoint); err == nil {
				t.Fatalf("expected an error, got %s:%d", addr, port)
			}
		})
	}
}

// warpRegisterRoundTripper serves the two calls RegisterWarp makes: the
// registration POST and the follow-up device GET.
type warpRegisterRoundTripper struct {
	peerEndpoint map[string]interface{}
	peerKey      string
}

func (r *warpRegisterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reply := func(payload interface{}) (*http.Response, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Header:     http.Header{},
		}, nil
	}

	if req.Method == http.MethodPost {
		return reply(map[string]interface{}{
			"id":      "device-id",
			"token":   "access-token",
			"account": map[string]interface{}{"license": "license-key"},
		})
	}
	return reply(map[string]interface{}{
		"config": map[string]interface{}{
			"interface": map[string]interface{}{
				"addresses": map[string]interface{}{
					"v4": "172.16.0.2",
					"v6": "2606:4700:110:8a1b:c1a2:d3b4:e5c6:f7d8",
				},
			},
			"peers": []interface{}{
				map[string]interface{}{
					"public_key": r.peerKey,
					"endpoint":   r.peerEndpoint,
				},
			},
		},
	})
}

// TestRegisterWarpAcceptsDomainEndpoint is the regression test for the save
// failure: registering a WARP endpoint returned "save: invalid warp peer
// endpoint" because Cloudflare's `endpoint.host` is a domain.
func TestRegisterWarpAcceptsDomainEndpoint(t *testing.T) {
	peerKey, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	oldClient := warpHTTPClient
	warpHTTPClient = &http.Client{Transport: &warpRegisterRoundTripper{
		peerKey: peerKey.PublicKey().String(),
		peerEndpoint: map[string]interface{}{
			"v4":    "162.159.192.1:0",
			"v6":    "[2606:4700:d0::a29f:c001]:0",
			"host":  "engage.cloudflareclient.com:2408",
			"ports": []interface{}{float64(2408), float64(500)},
		},
	}}
	t.Cleanup(func() { warpHTTPClient = oldClient })

	ep := &model.Endpoint{Options: json.RawMessage(`{"reserved":[0,0,0]}`)}
	if err := (&WarpService{}).RegisterWarp(ep); err != nil {
		t.Fatalf("RegisterWarp failed: %v", err)
	}

	var options struct {
		PrivateKey string   `json:"private_key"`
		Address    []string `json:"address"`
		ListenPort *int     `json:"listen_port"`
		Reserved   []int    `json:"reserved"`
		Peers      []struct {
			Address    string   `json:"address"`
			Port       int      `json:"port"`
			PublicKey  string   `json:"public_key"`
			AllowedIPs []string `json:"allowed_ips"`
		} `json:"peers"`
	}
	if err := json.Unmarshal(ep.Options, &options); err != nil {
		t.Fatal(err)
	}
	if len(options.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(options.Peers))
	}
	peer := options.Peers[0]
	if peer.Address != "162.159.192.1" {
		t.Fatalf("unexpected peer address: %q", peer.Address)
	}
	if peer.Port != 2408 {
		t.Fatalf("unexpected peer port: %d", peer.Port)
	}
	if peer.PublicKey != peerKey.PublicKey().String() {
		t.Fatalf("unexpected peer public key: %q", peer.PublicKey)
	}
	if len(peer.AllowedIPs) != 2 {
		t.Fatalf("unexpected allowed_ips: %#v", peer.AllowedIPs)
	}
	if options.Reserved != nil {
		t.Fatalf("reserved should be dropped, got %#v", options.Reserved)
	}
	if _, err := wgtypes.ParseKey(options.PrivateKey); err != nil {
		t.Fatalf("invalid generated private key: %v", err)
	}
	wantAddresses := []string{"172.16.0.2/32", "2606:4700:110:8a1b:c1a2:d3b4:e5c6:f7d8/128"}
	if len(options.Address) != 2 || options.Address[0] != wantAddresses[0] || options.Address[1] != wantAddresses[1] {
		t.Fatalf("unexpected addresses: %#v", options.Address)
	}

	var ext map[string]string
	if err := json.Unmarshal(ep.Ext, &ext); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"access_token": "access-token",
		"device_id":    "device-id",
		"license_key":  "license-key",
	} {
		if ext[key] != want {
			t.Fatalf("ext[%s] = %q, want %q", key, ext[key], want)
		}
	}
	if ext["api_version"] == "" {
		t.Fatal("ext.api_version should be recorded")
	}
}

// TestRegisterWarpHostOnlyEndpoint covers API versions that return only the
// domain form, with no v4/v6 addresses at all.
func TestRegisterWarpHostOnlyEndpoint(t *testing.T) {
	peerKey, err := wgtypes.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	oldClient := warpHTTPClient
	warpHTTPClient = &http.Client{Transport: &warpRegisterRoundTripper{
		peerKey:      peerKey.PublicKey().String(),
		peerEndpoint: map[string]interface{}{"host": "engage.cloudflareclient.com:2408"},
	}}
	t.Cleanup(func() { warpHTTPClient = oldClient })

	ep := &model.Endpoint{Options: json.RawMessage(`{}`)}
	if err := (&WarpService{}).RegisterWarp(ep); err != nil {
		t.Fatalf("RegisterWarp failed: %v", err)
	}

	var options struct {
		Peers []struct {
			Address string `json:"address"`
			Port    int    `json:"port"`
		} `json:"peers"`
	}
	if err := json.Unmarshal(ep.Options, &options); err != nil {
		t.Fatal(err)
	}
	if len(options.Peers) != 1 || options.Peers[0].Address != "engage.cloudflareclient.com" || options.Peers[0].Port != 2408 {
		t.Fatalf("unexpected peers: %#v", options.Peers)
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
