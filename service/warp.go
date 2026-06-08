package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util/common"

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
func doWarpRequestVersions(mkRequest func(version string) (*http.Request, []byte, error)) (*http.Response, string, error) {
	const attemptsPerVersion = 3
	var lastErr error
	for _, version := range warpAPIVersions {
		for attempt := 1; attempt <= attemptsPerVersion; attempt++ {
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
				// 4xx / 5xx — no point retrying within the same version,
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
				time.Sleep(time.Duration(attempt) * time.Second)
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
	privateKey, _ := wgtypes.GenerateKey()
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

	resp, version, err := doWarpRequestVersions(func(version string) (*http.Request, []byte, error) {
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

	deviceId, ok := rspData["id"].(string)
	if !ok {
		return common.NewError("missing warp device id")
	}
	token, ok := rspData["token"].(string)
	if !ok {
		return common.NewError("missing warp token")
	}
	account, ok := rspData["account"].(map[string]interface{})
	if !ok {
		return common.NewError("missing warp account")
	}
	license, ok := account["license"].(string)
	if !ok {
		logger.Debug("Error accessing license value.")
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

	warpConfig, _ := warpDetails["config"].(map[string]interface{})
	clientId, _ := warpConfig["client_id"].(string)
	reserved := s.getReserved(clientId)
	interfaceConfig, _ := warpConfig["interface"].(map[string]interface{})
	addresses, _ := interfaceConfig["addresses"].(map[string]interface{})
	v4, _ := addresses["v4"].(string)
	v6, _ := addresses["v6"].(string)
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
	peerEndpoint, ok := peerEndpointObj["host"].(string)
	if !ok {
		return common.NewError("missing warp peer endpoint host")
	}
	peerEpAddress, peerEpPort, err := net.SplitHostPort(peerEndpoint)
	if err != nil {
		return err
	}
	peerPublicKey, _ := peer["public_key"].(string)
	peerPort, _ := strconv.Atoi(peerEpPort)

	peerConfigs := []map[string]interface{}{
		{
			"address":     peerEpAddress,
			"port":        peerPort,
			"public_key":  peerPublicKey,
			"allowed_ips": []string{"0.0.0.0/0", "::/0"},
			"reserved":    reserved,
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

	ep.Options, err = json.MarshalIndent(epOptions, "", "  ")
	return err
}

func (s *WarpService) getReserved(clientID string) []int {
	var reserved []int
	decoded, err := base64.StdEncoding.DecodeString(clientID)
	if err != nil {
		return nil
	}

	hexString := ""
	for _, char := range decoded {
		hex := fmt.Sprintf("%02x", char)
		hexString += hex
	}

	for i := 0; i < len(hexString); i += 2 {
		hexByte := hexString[i : i+2]
		decValue, err := strconv.ParseInt(hexByte, 16, 32)
		if err != nil {
			return nil
		}
		reserved = append(reserved, int(decValue))
	}

	return reserved
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

	if success, ok := response["success"].(bool); ok && !success {
		errorArr, _ := response["errors"].([]interface{})
		if len(errorArr) == 0 {
			return common.NewError("warp license update failed")
		}
		errorObj, _ := errorArr[0].(map[string]interface{})
		return common.NewError(errorObj["code"], errorObj["message"])
	}

	return nil
}
