package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// makeSelfSignedCertPEM returns a self-signed leaf certificate PEM and its EC
// private key PEM, with the given NotAfter. Used to exercise the PEM-parsing and
// file-writing paths without contacting an ACME server.
func makeSelfSignedCertPEM(t *testing.T, notAfter time.Time) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "203.0.113.7"},
		NotBefore:    notAfter.Add(-160 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestParseCertNotAfter(t *testing.T) {
	want := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	certPEM, _ := makeSelfSignedCertPEM(t, want)
	got, err := parseCertNotAfter(certPEM)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Fatalf("parseCertNotAfter = %v, want %v", got, want)
	}

	if _, err := parseCertNotAfter([]byte("not a pem")); err == nil {
		t.Fatal("parseCertNotAfter on garbage = nil, want error")
	}

	// A PEM that contains a non-CERTIFICATE block first (e.g. a private key)
	// must be skipped so the leaf certificate's NotAfter is still found.
	certPEM2, keyPEM := makeSelfSignedCertPEM(t, want)
	withLeadingKey := append(append([]byte{}, keyPEM...), certPEM2...)
	got, err = parseCertNotAfter(withLeadingKey)
	if err != nil {
		t.Fatalf("parseCertNotAfter with leading key block: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("parseCertNotAfter (leading key) = %v, want %v", got, want)
	}

	// In a multi-certificate bundle the FIRST (leaf) certificate wins.
	leafWant := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	issuerWant := time.Date(2027, 1, 1, 9, 0, 0, 0, time.UTC)
	leafPEM, _ := makeSelfSignedCertPEM(t, leafWant)
	issuerPEM, _ := makeSelfSignedCertPEM(t, issuerWant)
	bundle := append(append([]byte{}, leafPEM...), issuerPEM...)
	got, err = parseCertNotAfter(bundle)
	if err != nil {
		t.Fatalf("parseCertNotAfter on bundle: %v", err)
	}
	if !got.Equal(leafWant) {
		t.Fatalf("parseCertNotAfter (bundle) = %v, want leaf %v", got, leafWant)
	}

	// A PEM with only a non-CERTIFICATE block yields an explicit error.
	if _, err := parseCertNotAfter(keyPEM); err == nil {
		t.Fatal("parseCertNotAfter on key-only PEM = nil, want error")
	}
}

func TestWriteCertFiles(t *testing.T) {
	t.Setenv("SUI_DB_FOLDER", t.TempDir())
	certPEM, keyPEM := makeSelfSignedCertPEM(t, time.Now().Add(160*time.Hour))

	certPath, keyPath, err := writeCertFiles("2001:db8::1", certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	// IPv6 colons must be sanitized out of the filename.
	if filepath.Base(certPath) != "ip-2001_db8__1.crt" {
		t.Fatalf("unexpected cert filename: %s", filepath.Base(certPath))
	}
	gotCert, err := os.ReadFile(certPath)
	if err != nil || string(gotCert) != string(certPEM) {
		t.Fatalf("cert file content mismatch: err=%v", err)
	}
	gotKey, err := os.ReadFile(keyPath)
	if err != nil || string(gotKey) != string(keyPEM) {
		t.Fatalf("key file content mismatch: err=%v", err)
	}

	// Key permissions are 0600 on POSIX (Windows ignores the mode bits).
	if runtime.GOOS != "windows" {
		info, err := os.Stat(keyPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("key perm = %o, want 600", info.Mode().Perm())
		}
	}

	if _, _, err := writeCertFiles("1.2.3.4", nil, keyPEM); err == nil {
		t.Fatal("writeCertFiles with empty cert = nil, want error")
	}
}
