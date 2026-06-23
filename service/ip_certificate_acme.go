package service

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// leShortlivedProfile is Let's Encrypt's certificate profile that authorises
// IP-address identifiers (RFC 8738). It yields ~160h (~6.7-day) certificates.
const leShortlivedProfile = "shortlived"

// acmeRequest is the input to an ACME issuance. AccountKeyPEM/RegistrationJSON
// carry a previously-persisted account so renewals reuse it instead of
// registering a fresh account every time.
type acmeRequest struct {
	IP               string
	Email            string
	ChallengePort    int
	AccountKeyPEM    string // empty => a new account key is generated
	RegistrationJSON string // empty => the account is (re-)registered
}

// acmeResult is the output of a successful issuance. CertPEM is the bundled
// leaf+issuer chain. AccountKeyPEM/RegistrationJSON are echoed back so the
// caller can persist the (possibly newly created) account.
type acmeResult struct {
	CertPEM          []byte
	KeyPEM           []byte
	AccountKeyPEM    string
	RegistrationJSON string
}

// acmeIssuer is the seam that keeps all network/Let's Encrypt code out of the
// unit-testable path. Tests inject a fake; production uses legoIssuer.
type acmeIssuer interface {
	Obtain(ctx context.Context, req acmeRequest) (acmeResult, error)
}

// acmeDirectoryURL returns the ACME directory endpoint. It defaults to Let's
// Encrypt production; SUI_ACME_DIR_URL overrides it for staging/Pebble smoke
// tests only (never surfaced in the UI).
func acmeDirectoryURL() string {
	if v := strings.TrimSpace(os.Getenv("SUI_ACME_DIR_URL")); v != "" {
		return v
	}
	return lego.LEDirectoryProduction
}

// ipCertUser implements registration.User for the ACME account.
type ipCertUser struct {
	email        string
	key          crypto.PrivateKey
	registration *registration.Resource
}

func (u *ipCertUser) GetEmail() string                        { return u.email }
func (u *ipCertUser) GetRegistration() *registration.Resource { return u.registration }
func (u *ipCertUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// legoIssuer is the real, network-backed acmeIssuer.
type legoIssuer struct{}

func (legoIssuer) Obtain(ctx context.Context, req acmeRequest) (acmeResult, error) {
	// lego's Certificate.Obtain has no context parameter, so an in-flight ACME
	// exchange cannot be cancelled; honour an already-cancelled context (e.g. a
	// shutdown or a disconnected API client) by refusing to start the issuance.
	if err := ctx.Err(); err != nil {
		return acmeResult{}, err
	}
	if err := validateIssuableIP(req.IP); err != nil {
		return acmeResult{}, err
	}

	user, accountKeyPEM, err := buildIpCertUser(req)
	if err != nil {
		return acmeResult{}, err
	}

	config := lego.NewConfig(user)
	config.CADirURL = acmeDirectoryURL()
	config.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(config)
	if err != nil {
		return acmeResult{}, err
	}

	port := req.ChallengePort
	if port <= 0 {
		port = 80
	}
	provider := http01.NewProviderServer("", strconv.Itoa(port))
	if err = client.Challenge.SetHTTP01Provider(provider); err != nil {
		return acmeResult{}, err
	}

	registrationJSON := req.RegistrationJSON
	if user.GetRegistration() == nil {
		reg, regErr := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if regErr != nil {
			return acmeResult{}, regErr
		}
		user.registration = reg
		marshalled, marshalErr := json.Marshal(reg)
		if marshalErr != nil {
			return acmeResult{}, marshalErr
		}
		registrationJSON = string(marshalled)
	}

	// Build the CSR ourselves so the IP lands in the subjectAltName with an
	// EMPTY Common Name. lego's Certificate.Obtain copies the single "domain"
	// into the CSR Subject.CommonName, which Let's Encrypt rejects for IP
	// identifiers with "badCSR :: CSR contains IP address in Common Name"
	// (RFC 8738 requires the IP only in the SAN). ObtainForCSR instead infers
	// the IP identifier from the CSR's IPAddresses SAN.
	leafKey, csr, err := buildIpCSR(req.IP)
	if err != nil {
		return acmeResult{}, err
	}

	resource, err := client.Certificate.ObtainForCSR(certificate.ObtainForCSRRequest{
		CSR:        csr,
		PrivateKey: leafKey,
		Profile:    leShortlivedProfile,
		Bundle:     true,
	})
	if err != nil {
		return acmeResult{}, err
	}
	if resource == nil || len(resource.Certificate) == 0 || len(resource.PrivateKey) == 0 {
		return acmeResult{}, common.NewError("ip cert: ACME returned an empty certificate")
	}

	return acmeResult{
		CertPEM:          resource.Certificate,
		KeyPEM:           resource.PrivateKey,
		AccountKeyPEM:    accountKeyPEM,
		RegistrationJSON: registrationJSON,
	}, nil
}

// buildIpCertUser loads or generates the ACME account key and decodes any
// stored registration. It returns the user and the PEM of the account key so
// the caller can persist a freshly generated one.
func buildIpCertUser(req acmeRequest) (*ipCertUser, string, error) {
	var (
		key           crypto.PrivateKey
		accountKeyPEM string
		err           error
	)
	if strings.TrimSpace(req.AccountKeyPEM) != "" {
		key, err = certcrypto.ParsePEMPrivateKey([]byte(req.AccountKeyPEM))
		if err != nil {
			return nil, "", common.NewError("ip cert: stored ACME account key is invalid: ", err.Error())
		}
		accountKeyPEM = req.AccountKeyPEM
	} else {
		key, err = certcrypto.GeneratePrivateKey(certcrypto.EC256)
		if err != nil {
			return nil, "", err
		}
		accountKeyPEM = string(certcrypto.PEMEncode(key))
	}

	user := &ipCertUser{email: req.Email, key: key}
	if strings.TrimSpace(req.RegistrationJSON) != "" {
		var reg registration.Resource
		if err = json.Unmarshal([]byte(req.RegistrationJSON), &reg); err != nil {
			return nil, "", common.NewError("ip cert: stored ACME registration is invalid: ", err.Error())
		}
		user.registration = &reg
	}
	return user, accountKeyPEM, nil
}

// buildIpCSR creates an EC256 leaf key and a CSR carrying the target IP as the
// sole subjectAltName with an EMPTY Subject Common Name. The empty CN is what
// keeps Let's Encrypt from rejecting the order with "CSR contains IP address in
// Common Name": for IP identifiers (RFC 8738) the address must appear only in
// the SAN. certcrypto.CreateCSR routes a SAN entry that parses as an IP into the
// CSR IPAddresses, and lego's ObtainForCSR derives the ACME "ip" identifier from
// there.
func buildIpCSR(ip string) (crypto.PrivateKey, *x509.CertificateRequest, error) {
	leafKey, err := certcrypto.GeneratePrivateKey(certcrypto.EC256)
	if err != nil {
		return nil, nil, err
	}
	csrDER, err := certcrypto.CreateCSR(leafKey, certcrypto.CSROptions{
		Domain: "", // empty Common Name - IP must not be the CN
		SAN:    []string{strings.TrimSpace(ip)},
	})
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, nil, err
	}
	return leafKey, csr, nil
}
