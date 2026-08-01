package service

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/config"
	"github.com/deposist/s-ui-x-extended/util/common"
)

// managedCertDir is where issued IP certificates live, under the panel data
// directory (next to the sqlite db). config.GetDBFolderPath honours
// SUI_DB_FOLDER and falls back to <exedir>/db.
func managedCertDir() string {
	return filepath.Join(config.GetDBFolderPath(), "certs")
}

// sanitizeIPForFilename makes an IP literal safe as a filename component
// (IPv6 colons are illegal on Windows).
func sanitizeIPForFilename(ip string) string {
	r := strings.NewReplacer(":", "_", "/", "_", "%", "_")
	return r.Replace(strings.TrimSpace(ip))
}

// writeCertFiles writes a complete certificate generation into a private
// directory before returning either path. Callers publish the returned pair
// only after both writes succeed, so a failed key write can never expose a new
// certificate beside an older key.
func writeCertFiles(ip string, certPEM, keyPEM []byte) (certPath, keyPath string, err error) {
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return "", "", common.NewError("ip cert: empty certificate or key material")
	}
	dir := managedCertDir()
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return "", "", err
	}
	generationDir, err := os.MkdirTemp(dir, "generation-")
	if err != nil {
		return "", "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(generationDir)
		}
	}()
	base := "ip-" + sanitizeIPForFilename(ip)
	certPath = filepath.Join(generationDir, base+".crt")
	keyPath = filepath.Join(generationDir, base+".key")
	if err = os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return "", "", err
	}
	if err = os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return "", "", err
	}
	committed = true
	return certPath, keyPath, nil
}

// parseCertNotAfter reads the NotAfter of the leaf certificate from a PEM
// bundle (first CERTIFICATE block). Pure aside from no I/O.
func parseCertNotAfter(certPEM []byte) (time.Time, error) {
	rest := certPEM
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return time.Time{}, common.NewError("ip cert: no CERTIFICATE block in PEM")
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return time.Time{}, err
		}
		return cert.NotAfter, nil
	}
}
