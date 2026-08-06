package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/util/common"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type WarpService struct{}

// warpAPIVersions lists Cloudflare WARP REST API versions in the order we
// will try them. The newer `v0a4005` endpoint is what current first-party
// clients (1.1.1.1 desktop / wgcf) speak; the older `v0a2158` endpoint
// occasionally still works and is kept as a fallback for hosts where the
// new endpoint refuses the connection.
var warpAPIVersions = []string{"v0a4005", "v0a2158"}

// warpUserAgent mimics a current 1.1.1.1 desktop client. Without this header
// Cloudflare regularly drops the TLS connection mid-stream (`EOF`) before
// returning a body.
const warpUserAgent = "1.1.1.1/6.81"

// warpClientVersion mirrors the matching CF-Client-Version a recent first
// party client sends.
const warpClientVersion = "a-6.81-3343"

// warpHTTPClient is the dedicated client used for Cloudflare WARP API
// calls. The Cloudflare endpoint is fussy about TLS minor versions and
// HTTP/2 multiplexing on slow uplinks, so we pin TLS 1.2+ and stay on
// HTTP/1.1.
var warpHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:   false,
		TLSNextProto:        map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConns:        4,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// setWarpHeaders applies the headers a current first-party WARP client
// sends. Cloudflare uses these to distinguish trusted clients from generic
// HTTP clients; without them registration requests are met with `EOF`.
func setWarpHeaders(req *http.Request) {
	req.Header.Set("User-Agent", warpUserAgent)
	req.Header.Set("CF-Client-Version", warpClientVersion)
	req.Header.Set("Accept", "application/json; charset=UTF-8")
	req.Header.Set("Accept-Encoding", "identity")
	if req.Method != http.MethodGet && req.Method != http.MethodDelete {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		}
	}
}

func setWarpAuthorizedHeaders(req *http.Request, accessToken string) {
	setWarpHeaders(req)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
}

// doWarpAttempt performs a single HTTP attempt with proper body cloning so
// retries can replay POSTs / PUTs.
func doWarpAttempt(req *http.Request, body []byte) (*http.Response, error) {
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	}
	return warpHTTPClient.Do(req)
}

// doWarpRequestVersions issues the same request against each WARP API
// version until one returns a 2xx response. The provided `mkRequest`
// callback rebuilds the request for a given version (the URL changes).
//
// Each version is retried up to 3 times to absorb transient TLS / network
// hiccups. The last error is preserved when all attempts fail.
func doWarpRequestVersions(ctx context.Context, mkRequest func(version string) (*http.Request, []byte, error)) (*http.Response, string, error) {
	const attemptsPerVersion = 3
	var lastErr error
	for _, version := range warpAPIVersions {
		for attempt := 1; attempt <= attemptsPerVersion; attempt++ {
			if err := ctx.Err(); err != nil {
				return nil, "", err
			}
			req, body, err := mkRequest(version)
			if err != nil {
				return nil, "", err
			}
			setWarpHeaders(req)
			resp, err := doWarpAttempt(req, body)
			if err == nil {
				if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
					return resp, version, nil
				}
				// 4xx / 5xx - no point retrying within the same version,
				// but the next version may behave differently.
				_ = resp.Body.Close()
				lastErr = common.NewErrorf("cloudflare warp %s status: %d", version, resp.StatusCode)
				logger.Warningf("warp request to %s returned %d, will try other versions", version, resp.StatusCode)
				break
			}
			lastErr = err
			logger.Warningf("warp request attempt %d/%d on %s failed: %v", attempt, attemptsPerVersion, version, err)
			// EOF / connection-reset are the most likely failure modes here;
			// a brief backoff helps Cloudflare recycle the trust window.
			if attempt < attemptsPerVersion {
				select {
				case <-time.After(time.Duration(attempt) * time.Second):
				case <-ctx.Done():
					return nil, "", ctx.Err()
				}
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("cloudflare warp: all attempts failed")
	}
	return nil, "", lastErr
}

func (s *WarpService) getWarpInfo(version, deviceId, accessToken string) ([]byte, error) {
	url := fmt.Sprintf("https://api.cloudflareclient.com/%s/reg/%s", version, deviceId)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setWarpAuthorizedHeaders(req, accessToken)
	resp, err := doWarpAttempt(req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, common.NewErrorf("cloudflare warp status: %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func (s *WarpService) RegisterWarp(ep *model.Endpoint) error {
	tos := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	privateKey, err := wgtypes.GenerateKey()
	if err != nil {
		return common.NewError("generate warp private key: ", err.Error())
	}
	publicKey := privateKey.PublicKey().String()
	hostName, _ := os.Hostname()

	dataBytes, err := json.Marshal(map[string]string{
		"key":    publicKey,
		"tos":    tos,
		"type":   "PC",
		"model":  "s-ui",
		"name":   hostName,
		"locale": "en_US",
	})
	if err != nil {
		return err
	}

	resp, version, err := doWarpRequestVersions(context.Background(), func(version string) (*http.Request, []byte, error) {
		url := fmt.Sprintf("https://api.cloudflareclient.com/%s/reg", version)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, nil)
		if err != nil {
			return nil, nil, err
		}
		return req, dataBytes, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var rspData map[string]interface{}
	if err := json.Unmarshal(body, &rspData); err != nil {
		return err
	}

	deviceId, ok := nonEmptyWarpString(rspData, "id")
	if !ok {
		return common.NewError("missing warp device id")
	}
	token, ok := nonEmptyWarpString(rspData, "token")
	if !ok {
		return common.NewError("missing warp token")
	}
	account, ok := rspData["account"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp account")
	}
	license, ok := nonEmptyWarpString(account, "license")
	if !ok {
		return common.NewError("missing warp license")
	}

	warpInfo, err := s.getWarpInfo(version, deviceId, token)
	if err != nil {
		return err
	}

	var warpDetails map[string]interface{}
	if err := json.Unmarshal(warpInfo, &warpDetails); err != nil {
		return err
	}

	warpConfig, ok := warpDetails["config"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp config")
	}
	interfaceConfig, ok := warpConfig["interface"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp interface")
	}
	addresses, ok := interfaceConfig["addresses"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp addresses")
	}
	v4, ok := nonEmptyWarpString(addresses, "v4")
	if !ok || net.ParseIP(v4) == nil {
		return common.NewError("invalid warp IPv4 address")
	}
	v6, ok := nonEmptyWarpString(addresses, "v6")
	if !ok || net.ParseIP(v6) == nil {
		return common.NewError("invalid warp IPv6 address")
	}
	peers, ok := warpConfig["peers"].([]interface{})
	if !ok || len(peers) == 0 {
		return common.NewError("missing warp peers")
	}
	peer, ok := peers[0].(map[string]interface{})
	if !ok {
		return common.NewError("invalid warp peer")
	}
	peerEndpointObj, ok := peer["endpoint"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp peer endpoint")
	}
	peerEpAddress, peerPort, err := parseWarpPeerEndpoint(peerEndpointObj)
	if err != nil {
		return err
	}
	peerPublicKey, ok := nonEmptyWarpString(peer, "public_key")
	if !ok {
		return common.NewError("missing warp peer public key")
	}
	if _, err := wgtypes.ParseKey(peerPublicKey); err != nil {
		return common.NewError("invalid warp peer public key")
	}

	peerConfigs := []map[string]interface{}{
		{
			"address":     peerEpAddress,
			"port":        peerPort,
			"public_key":  peerPublicKey,
			"allowed_ips": []string{"0.0.0.0/0", "::/0"},
		},
	}

	warpData := map[string]interface{}{
		"access_token": token,
		"device_id":    deviceId,
		"license_key":  license,
		"api_version":  version,
	}

	ep.Ext, err = json.MarshalIndent(warpData, "", "  ")
	if err != nil {
		return err
	}

	var epOptions map[string]interface{}
	if err := json.Unmarshal(ep.Options, &epOptions); err != nil {
		return err
	}
	epOptions["private_key"] = privateKey.String()
	epOptions["address"] = []string{fmt.Sprintf("%s/32", v4), fmt.Sprintf("%s/128", v6)}
	epOptions["listen_port"] = 0
	epOptions["peers"] = peerConfigs
	delete(epOptions, "reserved")

	ep.Options, err = json.MarshalIndent(epOptions, "", "  ")
	return err
}

func nonEmptyWarpString(values map[string]interface{}, key string) (string, bool) {
	value, ok := values[key].(string)
	return value, ok && value != ""
}

// warpDefaultPeerPort is the port first-party WARP clients use. It is the
// fallback for the `v4` / `v6` endpoint forms, which Cloudflare returns with
// a `:0` placeholder port rather than a real one.
const warpDefaultPeerPort = 2408

// parseWarpPeerEndpoint extracts a peer address and port from a WARP
// registration `peers[].endpoint` object.
//
// Cloudflare returns the same endpoint in several forms, and which of them
// are populated varies between API versions:
//
//	"endpoint": {
//	  "v4":    "162.159.192.1:0",
//	  "v6":    "[2606:4700:d0::a29f:c001]:0",
//	  "host":  "engage.cloudflareclient.com:2408",
//	  "ports": [2408, 500, 1701, 4500]
//	}
//
// Two properties of that payload matter here. `host` is a DOMAIN, not an IP,
// so an IP-only check rejects every real response. And the `v4` / `v6` forms
// carry port 0, which is a placeholder, so a port from `ports` (or the
// well-known 2408) is substituted instead.
//
// The IP forms are preferred over `host`: a literal address keeps the core
// from having to resolve the peer through its own DNS at start-up, which is
// a bootstrap hazard when DNS is itself routed over the tunnel. `host` stays
// as a fallback for versions that omit the addresses, and sing-box resolves
// it via `ResolvePeer` when it is a domain.
func parseWarpPeerEndpoint(endpoint map[string]interface{}) (string, int, error) {
	fallbackPort := warpDefaultPeerPort
	if ports, ok := endpoint["ports"].([]interface{}); ok {
		for _, entry := range ports {
			port, ok := entry.(float64)
			if ok && port >= 1 && port <= 65535 {
				fallbackPort = int(port)
				break
			}
		}
	}

	var lastErr error
	for _, key := range []string{"v4", "host", "v6"} {
		raw, ok := nonEmptyWarpString(endpoint, key)
		if !ok {
			continue
		}
		address, portString, err := net.SplitHostPort(raw)
		if err != nil {
			// Some responses carry the address without a port at all.
			address, portString = raw, ""
		}
		address = strings.Trim(address, "[]")
		if address == "" || (net.ParseIP(address) == nil && !isWarpPeerHostname(address)) {
			lastErr = common.NewErrorf("invalid warp peer endpoint %q", raw)
			continue
		}
		port := fallbackPort
		if portString != "" {
			parsed, err := strconv.Atoi(portString)
			if err != nil {
				lastErr = common.NewErrorf("invalid warp peer endpoint port %q", raw)
				continue
			}
			// Anything outside the valid range (in practice the `:0`
			// placeholder) falls back to a usable port.
			if parsed >= 1 && parsed <= 65535 {
				port = parsed
			}
		}
		return address, port, nil
	}
	if lastErr != nil {
		return "", 0, lastErr
	}
	return "", 0, common.NewError("missing warp peer endpoint host")
}

// isWarpPeerHostname reports whether host looks like a DNS name. It is
// deliberately lenient about TLDs - the value comes from Cloudflare, so the
// only goal is rejecting junk that would produce a broken core config.
func isWarpPeerHostname(host string) bool {
	host = strings.TrimSuffix(host, ".")
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if r > 127 || !(r == '-' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return false
			}
		}
	}
	return true
}

func uniqueWarpAPIVersions(preferred string) []string {
	versions := make([]string, 0, len(warpAPIVersions)+1)
	seen := make(map[string]struct{}, len(warpAPIVersions)+1)
	add := func(version string) {
		if version == "" {
			return
		}
		if _, ok := seen[version]; ok {
			return
		}
		seen[version] = struct{}{}
		versions = append(versions, version)
	}
	add(preferred)
	for _, version := range warpAPIVersions {
		add(version)
	}
	return versions
}

func (s *WarpService) SetWarpLicense(old_license string, ep *model.Endpoint) error {
	var warpData map[string]string
	if err := json.Unmarshal(ep.Ext, &warpData); err != nil {
		return err
	}

	if warpData["license_key"] == old_license {
		return nil
	}

	dataBytes, err := json.Marshal(map[string]string{"license": warpData["license_key"]})
	if err != nil {
		return err
	}

	// Prefer the API version captured during registration; fall back to
	// trying every version if it is missing or stops working.
	versions := uniqueWarpAPIVersions(warpData["api_version"])

	var resp *http.Response
	var lastErr error
attempt:
	for _, version := range versions {
		url := fmt.Sprintf("https://api.cloudflareclient.com/%s/reg/%s/account", version, warpData["device_id"])
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, nil)
		if err != nil {
			return err
		}
		setWarpAuthorizedHeaders(req, warpData["access_token"])
		r, err := doWarpAttempt(req, dataBytes)
		if err != nil {
			lastErr = err
			logger.Warningf("warp license update on %s failed: %v", version, err)
			continue
		}
		if r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices {
			resp = r
			break attempt
		}
		_ = r.Body.Close()
		lastErr = common.NewErrorf("cloudflare warp %s status: %d", version, r.StatusCode)
		logger.Warningf("warp license update on %s returned %d", version, r.StatusCode)
	}
	if resp == nil {
		if lastErr == nil {
			lastErr = errors.New("cloudflare warp: all attempts failed")
		}
		return lastErr
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	success, ok := response["success"].(bool)
	if !ok {
		return common.NewError("warp license update returned invalid success status")
	}
	if !success {
		errorArr, _ := response["errors"].([]interface{})
		if len(errorArr) == 0 {
			return common.NewError("warp license update failed")
		}
		errorObj, ok := errorArr[0].(map[string]interface{})
		if !ok {
			return common.NewError("warp license update failed")
		}
		return common.NewError(errorObj["code"], errorObj["message"])
	}

	return nil
}
