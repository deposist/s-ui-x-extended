package service

import (
	"net"
	"testing"
)

// TestBuildIpCSRHasEmptyCommonNameAndIpSAN locks in the badCSR fix: the CSR sent
// to Let's Encrypt for an IP identifier must carry the IP only in the
// subjectAltName, with an EMPTY Common Name. A non-empty CN containing the IP is
// exactly what triggers "badCSR :: CSR contains IP address in Common Name".
func TestBuildIpCSRHasEmptyCommonNameAndIpSAN(t *testing.T) {
	const ip = "93.184.216.34"
	key, csr, err := buildIpCSR(ip)
	if err != nil {
		t.Fatal(err)
	}
	if key == nil {
		t.Fatal("buildIpCSR returned a nil leaf key")
	}
	if csr.Subject.CommonName != "" {
		t.Fatalf("CommonName = %q, want empty (IP in CN is rejected by Let's Encrypt)", csr.Subject.CommonName)
	}
	if len(csr.IPAddresses) != 1 || !csr.IPAddresses[0].Equal(net.ParseIP(ip)) {
		t.Fatalf("IPAddresses = %v, want [%s]", csr.IPAddresses, ip)
	}
	if len(csr.DNSNames) != 0 {
		t.Fatalf("DNSNames = %v, want none for an IP certificate", csr.DNSNames)
	}
	// lego forwards csr.Raw verbatim to the finalize step, so the parsed CSR must
	// carry a valid self-signature.
	if err := csr.CheckSignature(); err != nil {
		t.Fatalf("CSR signature invalid: %v", err)
	}
}

// TestBuildIpCSRTrimsWhitespace guards against a padded IP literal leaking into
// the SAN (which would make the SAN value mismatch the ACME identifier).
func TestBuildIpCSRTrimsWhitespace(t *testing.T) {
	_, csr, err := buildIpCSR("  93.184.216.34 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(csr.IPAddresses) != 1 || !csr.IPAddresses[0].Equal(net.ParseIP("93.184.216.34")) {
		t.Fatalf("IPAddresses = %v, want trimmed [93.184.216.34]", csr.IPAddresses)
	}
}
