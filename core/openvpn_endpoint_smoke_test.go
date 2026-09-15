//go:build with_openvpn

package core

// Real OpenVPN client<->server TLS tunnel between two panel-composed Box
// instances, with proxied HTTP traffic as the observable.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sagernet/sing-box/option"
)

type ovpnCerts struct {
	caPath, serverCert, serverKey, clientCert, clientKey string
}

func ovpnWrite(t *testing.T, dir, name string, blocks ...*pem.Block) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, b := range blocks {
		if err := pem.Encode(f, b); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func ovpnCertsFor(t *testing.T) ovpnCerts {
	t.Helper()
	dir := t.TempDir()
	newKey := func() *rsa.PrivateKey {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	serial := func() *big.Int {
		n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	caKey := newKey()
	caTmpl := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "ovpn-smoke-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	issue := func(cn string, eku x509.ExtKeyUsage) (string, string) {
		key := newKey()
		tmpl := &x509.Certificate{
			SerialNumber: serial(),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  []x509.ExtKeyUsage{eku},
		}
		der, cerr := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		if cerr != nil {
			t.Fatal(cerr)
		}
		certPath := ovpnWrite(t, dir, cn+".crt", &pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyPath := ovpnWrite(t, dir, cn+".key", &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
		return certPath, keyPath
	}
	caPath := ovpnWrite(t, dir, "ca.crt", &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	serverCert, serverKey := issue("ovpn-server", x509.ExtKeyUsageServerAuth)
	clientCert, clientKey := issue("ovpn-client", x509.ExtKeyUsageClientAuth)
	return ovpnCerts{caPath: caPath, serverCert: serverCert, serverKey: serverKey, clientCert: clientCert, clientKey: clientKey}
}

func ovpnStartBox(t *testing.T, cfg string) *Box {
	t.Helper()
	ctx := Context(context.Background(), InboundRegistry(), OutboundRegistry(), EndpointRegistry(), ProviderRegistry(), DNSTransportRegistry(), ServiceRegistry(), CertificateProviderRegistry())
	var opt option.Options
	if err := opt.UnmarshalJSONContext(ctx, []byte(cfg)); err != nil {
		t.Fatalf("unmarshal box config: %v", err)
	}
	box, err := NewBox(Options{Context: ctx, Options: opt})
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	if err := box.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return box
}

func ovpnHostIPv4s(t *testing.T) []net.IP {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	var out []net.IP
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			continue
		}
		out = append(out, ip)
	}
	if len(out) == 0 {
		t.Fatal("no bindable non-loopback IPv4 address")
	}
	return out
}

func ovpnFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// TestOpenVPNTunnelSmoke starts a real OpenVPN server endpoint and a real
// OpenVPN client endpoint in two panel-composed boxes (TLS mode, generated
// certificates, user/password auth) and proves HTTP traffic travels through the
// tunnel: the socks client asks the client box for an origin address only the
// server box can reach, and the response comes back through the tunnel.
//
// The origin must listen on a real interface address: the receiving box drops
// martian packets whose destination is a loopback address, so 127.0.0.1 is not
// reachable across the tunnel by design.
func TestOpenVPNTunnelSmoke(t *testing.T) {
	certs := ovpnCertsFor(t)
	originPort := ovpnFreePort(t)
	socksPort := ovpnFreePort(t)
	serverPort := ovpnFreePort(t)
	const payload = "OVPN-TUNNEL-OK"

	// The tunnel delivers packets to the server box, which drops loopback
	// destinations as martians; use a real interface address instead.
	var hostIP net.IP
	var listener net.Listener
	var err error
	for _, candidate := range ovpnHostIPv4s(t) {
		listener, err = net.Listen("tcp", net.JoinHostPort(candidate.String(), strconv.Itoa(originPort)))
		if err == nil {
			hostIP = candidate
			break
		}
	}
	if hostIP == nil {
		t.Fatalf("cannot bind origin to any interface address: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, aerr := listener.Accept()
			if aerr != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, _ = c.Read(buf)
				_, _ = io.WriteString(c, "HTTP/1.0 200 OK\r\nContent-Length: "+fmt.Sprint(len(payload))+"\r\n\r\n"+payload)
			}(conn)
		}
	}()

	serverCfg := fmt.Sprintf(`{
		"log":{"disabled":true},
		"endpoints":[{"type":"openvpn-server","tag":"ovpn-srv","listen":"127.0.0.1","listen_port":%d,
			"network":"udp","address":["10.88.0.1/24"],
			"users":[{"username":"smoke","password":"smokepass"}],
			"tls":{"certificate_path":%q,"key_path":%q,"client_certificate_path":%q}}],
		"outbounds":[{"type":"direct","tag":"direct"}],
		"route":{"final":"direct"}
	}`, serverPort, certs.serverCert, certs.serverKey, certs.caPath)

	clientCfg := fmt.Sprintf(`{
		"log":{"disabled":true},
		"endpoints":[{"type":"openvpn-client","tag":"ovpn-cli","network":"udp","server":"127.0.0.1","server_port":%d,
			"username":"smoke","password":"smokepass",
			"tls":{"certificate_path":%q,"client_certificate_path":%q,"client_key_path":%q}}],
		"inbounds":[{"type":"socks","tag":"socks-in","listen":"127.0.0.1","listen_port":%d}],
		"route":{"final":"ovpn-cli"}
	}`, serverPort, certs.caPath, certs.clientCert, certs.clientKey, socksPort)

	serverBox := ovpnStartBox(t, serverCfg)
	defer serverBox.Close()
	clientBox := ovpnStartBox(t, clientCfg)
	defer clientBox.Close()

	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, derr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), 3*time.Second)
		if derr != nil {
			lastErr = derr
			time.Sleep(time.Second)
			continue
		}
		_, _ = conn.Write([]byte{0x05, 0x01, 0x00})
		greet := make([]byte, 2)
		if _, derr = io.ReadFull(conn, greet); derr != nil || greet[0] != 0x05 {
			lastErr = fmt.Errorf("greeting: %v %v", derr, greet)
			conn.Close()
			time.Sleep(time.Second)
			continue
		}
		req := []byte{0x05, 0x01, 0x00, 0x01, hostIP[0], hostIP[1], hostIP[2], hostIP[3], byte(originPort >> 8), byte(originPort)}
		_, _ = conn.Write(req)
		reply := make([]byte, 10)
		if _, derr = io.ReadFull(conn, reply); derr != nil || reply[1] != 0x00 {
			lastErr = fmt.Errorf("socks connect reply: %v %v", derr, reply)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		_, _ = io.WriteString(conn, "GET / HTTP/1.0\r\nHost: "+hostIP.String()+"\r\n\r\n")
		_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		body, rerr := io.ReadAll(conn)
		conn.Close()
		lastErr = rerr
		if rerr == nil && len(body) > 0 {
			text := string(body)
			if strings.Contains(text, payload) {
				t.Logf("openvpn tunnel OK: %q", payload)
				return
			}
			lastErr = fmt.Errorf("unexpected body: %q", text)
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("openvpn tunnel smoke failed: %v", lastErr)
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
