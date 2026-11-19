// Command nodes-panel provisions disposable, loopback-only HTTPS panels for Nodes browser tests.
package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/app"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/common/metadata"
)

func main() {
	switch {
	case len(os.Args) == 3 && os.Args[1] == "cert":
		if err := createCertificate(os.Args[2]); err != nil {
			log.Fatal(err)
		}
	case len(os.Args) == 6 && os.Args[1] == "serve":
		if err := serve(os.Args[2], os.Args[3], os.Args[4], os.Args[5]); err != nil {
			log.Fatal(err)
		}
	case len(os.Args) == 2 && os.Args[1] == "check-vless":
		if !checkVLESSSubscription() {
			os.Exit(1)
		}
	default:
		log.Fatal("usage: nodes-panel cert DIR | serve CERT_DIR WEB_PORT SUB_PORT PANEL_PATH | check-vless")
	}
}

func checkVLESSSubscription() bool {
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 4096))
	if err != nil {
		return false
	}
	link, err := url.Parse(strings.TrimSpace(string(raw)))
	if err != nil || link.Scheme != "vless" || link.User == nil || link.Hostname() != "127.0.0.1" {
		return false
	}
	client, err := vless.NewClient(link.User.Username(), "", nil)
	if err != nil {
		return false
	}
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer endpoint.Close()
	conn, err := net.DialTimeout("tcp", link.Host, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return false
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint.URL, nil)
	if err != nil {
		return false
	}
	proxy, err := client.DialConn(conn, metadata.ParseSocksaddr(request.URL.Host))
	if err != nil || request.Write(proxy) != nil {
		return false
	}
	response, err := http.ReadResponse(bufio.NewReader(proxy), request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusNoContent
}

func createCertificate(dir string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	now := time.Now()
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Nodes loopback test"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}, BasicConstraintsValid: true, IsCA: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	rawCert, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return err
	}
	rawKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.WriteFile("panel.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rawCert}), 0o600); err != nil {
		return err
	}
	return root.WriteFile("panel.key", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: rawKey}), 0o600)
}

func serve(certDir, webPort, subPort, panelPath string) error {
	dbDir := os.Getenv("SUI_DB_FOLDER")
	if dbDir == "" {
		return errors.New("set SUI_DB_FOLDER to a disposable panel directory")
	}
	certPath := filepath.Join(certDir, "panel.pem")
	certRoot, err := os.OpenRoot(certDir)
	if err != nil {
		return err
	}
	defer certRoot.Close()
	root, err := certRoot.ReadFile("panel.pem")
	if err != nil {
		return err
	}
	trust := x509.NewCertPool()
	if !trust.AppendCertsFromPEM(root) {
		return errors.New("invalid loopback test certificate")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: trust, MinVersion: tls.VersionTLS12}
	http.DefaultTransport = transport
	panel := app.NewApp()
	if err := panel.Init(); err != nil {
		return err
	}
	dbRoot, err := os.OpenRoot(dbDir)
	if err != nil {
		return err
	}
	defer dbRoot.Close()
	password, err := dbRoot.ReadFile("initial-admin.txt")
	if err != nil {
		return err
	}
	if err := (&service.UserService{}).UpdateFirstUser("admin", strings.TrimSpace(string(password))); err != nil {
		return err
	}
	settings := map[string]string{
		"webListen": "127.0.0.1", "subListen": "127.0.0.1",
		"webPort": webPort, "subPort": subPort, "webPath": panelPath,
		"webCertFile": certPath, "webKeyFile": filepath.Join(certDir, "panel.key"),
	}
	for key, value := range settings {
		if err := database.GetDB().Where("key = ?", key).Assign(model.Setting{Value: value}).FirstOrCreate(&model.Setting{Key: key}).Error; err != nil {
			return err
		}
	}
	if err := panel.Start(); err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	<-ctx.Done()
	panel.Stop()
	return nil
}
