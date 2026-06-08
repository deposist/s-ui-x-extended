package util

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/util/common"
)

const maxExternalSubBytes = 4 << 20

var (
	externalHTTPClientOnce sync.Once
	externalHTTPClient     *http.Client
)

// errBlockedExternalAddress is returned by the dialer hook when the resolved
// IP address points at a private/loopback/etc. range while
// SUI_ALLOW_PRIVATE_SUB_URLS is not enabled.
var errBlockedExternalAddress = common.NewError("private url host is not allowed")

// allowPrivateExternalURLs reports whether SUI_ALLOW_PRIVATE_SUB_URLS opts the
// process out of private-address filtering for external subscription URLs.
func allowPrivateExternalURLs() bool {
	return os.Getenv("SUI_ALLOW_PRIVATE_SUB_URLS") == "true"
}

// getExternalHTTPClient returns a process-wide HTTP client that re-validates
// every dialed address against isBlockedExternalAddr. Re-validating at dial
// time prevents DNS-rebinding attacks where validateExternalURL sees a public
// address but the subsequent connection is steered to a private one.
func getExternalHTTPClient() *http.Client {
	externalHTTPClientOnce.Do(func() {
		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if allowPrivateExternalURLs() {
					return dialer.DialContext(ctx, network, addr)
				}
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				if addr, err := netip.ParseAddr(host); err == nil {
					if isBlockedExternalAddr(addr) {
						return nil, errBlockedExternalAddress
					}
					return dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil {
					return nil, err
				}
				var lastErr error
				for _, ip := range ips {
					addr, ok := netip.AddrFromSlice(ip.IP)
					if !ok || isBlockedExternalAddr(addr) {
						lastErr = errBlockedExternalAddress
						continue
					}
					conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
					if err == nil {
						return conn, nil
					}
					lastErr = err
				}
				if lastErr == nil {
					lastErr = errBlockedExternalAddress
				}
				return nil, lastErr
			},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
		}
		externalHTTPClient = &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		}
	})
	return externalHTTPClient
}

func GetExternalLink(rawURL string) (string, error) {
	if err := validateExternalURL(rawURL); err != nil {
		logger.Warning("sub: invalid external URL:", err)
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	response, err := getExternalHTTPClient().Do(req)
	if err != nil {
		if errors.Is(err, errBlockedExternalAddress) {
			logger.Warning("sub: external URL resolves to blocked address:", err)
			return "", err
		}
		logger.Warning("sub: Error making HTTP request:", err)
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", common.NewErrorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxExternalSubBytes+1))
	if err != nil {
		logger.Warning("sub: Error reading response body:", err)
		return "", err
	}
	if len(body) > maxExternalSubBytes {
		return "", common.NewError("response is too large")
	}

	data := StrOrBase64Encoded(string(body))
	return data, nil
}

func GetExternalSub(url string) ([]map[string]interface{}, error) {
	var err error
	var result []map[string]interface{}

	if len(url) == 0 {
		return nil, common.NewError("no url")
	}

	data, err := GetExternalLink(url)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, common.NewError("no result")
	}

	// if the data is a JSON object
	if strings.HasPrefix(data, "{") && strings.HasSuffix(data, "}") {
		var jsonData map[string]interface{}
		err = json.Unmarshal([]byte(data), &jsonData)
		if err != nil {
			logger.Warning("sub: Error unmarshalling JSON:", err)
			return nil, err
		}
		outbounds, ok := jsonData["outbounds"].([]any)
		if !ok {
			logger.Warning("sub: missing outbounds field")
			return nil, common.NewError("invalid subscription: missing outbounds")
		}
		for _, outbound := range outbounds {
			outboundMap, ok := outbound.(map[string]interface{})
			if ok && len(outboundMap) > 0 {
				oType, _ := outboundMap["type"].(string)
				switch oType {
				case "urltest", "direct", "selector", "block":
					continue
				default:
					result = append(result, outboundMap)
				}
			}
		}
		if len(result) == 0 {
			return nil, common.NewError("no result")
		}
		return result, nil
	}
	// if data is a text
	links := strings.Split(data, "\n")
	for _, link := range links {
		linkToJson, _, err := GetOutbound(link, 0)
		if err == nil {
			result = append(result, *linkToJson)
		}
	}
	if len(result) == 0 {
		return nil, common.NewError("no result")
	}
	return result, nil
}

func validateExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return common.NewError("unsupported url scheme")
	}
	host := parsed.Hostname()
	if host == "" {
		return common.NewError("missing url host")
	}
	if strings.EqualFold(host, "localhost") {
		return common.NewError("localhost url is not allowed")
	}
	if allowPrivateExternalURLs() {
		return nil
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if isBlockedExternalAddr(addr) {
			return common.NewError("private url host is not allowed")
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	if len(addrs) == 0 {
		return common.NewError("url host did not resolve")
	}
	for _, ipAddr := range addrs {
		addr, ok := netip.AddrFromSlice(ipAddr.IP)
		if !ok || isBlockedExternalAddr(addr) {
			return common.NewError("private url host is not allowed")
		}
	}
	return nil
}

func isBlockedExternalAddr(addr netip.Addr) bool {
	return addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() || addr.IsUnspecified()
}
