package importxui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/gorm"
)

// flexStringList unmarshals an Xray PEM field that may be a single string or a
// list of strings — both forms appear in tlsSettings.certificates entries.
type flexStringList []string

func (f *flexStringList) UnmarshalJSON(b []byte) error {
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		if strings.TrimSpace(single) != "" {
			*f = flexStringList{single}
		}
		return nil
	}
	var list []string
	if err := json.Unmarshal(b, &list); err == nil {
		*f = flexStringList(list)
	}
	// Any other shape is ignored: a missing/odd certificate just yields no spec.
	return nil
}

// xuiCertificate is one entry of an Xray tlsSettings.certificates array. The
// panel stores either inline PEM (certificate/key) or on-disk paths
// (certificateFile/keyFile).
type xuiCertificate struct {
	CertificateFile string         `json:"certificateFile"`
	KeyFile         string         `json:"keyFile"`
	Certificate     flexStringList `json:"certificate"`
	Key             flexStringList `json:"key"`
}

type tlsCertSpec struct {
	Key         string
	Name        string
	ServerName  string
	Certificate []string
	KeyPEM      []string
	ALPN        []string
	Insecure    bool
}

// extractPlainTLS builds a TLS spec from an inbound whose stream security is
// plain "tls" and whose tlsSettings carry an INLINE certificate+key. Only inline
// material can migrate from a database-only import: a certificate referenced by
// file path lives on the source host's disk, which the importer cannot read, so
// that case returns a warning and no spec.
func extractPlainTLS(row xuiInboundRow) (*tlsCertSpec, []string, error) {
	stream, err := parseStreamSettings(row)
	if err != nil {
		return nil, nil, err
	}
	if stream.Security != "tls" {
		return nil, nil, nil
	}
	t := stream.TLSSettings
	var certPEM, keyPEM []string
	pathOnly := false
	for _, c := range t.Certificates {
		if len(c.Certificate) > 0 && len(c.Key) > 0 {
			certPEM = []string(c.Certificate)
			keyPEM = []string(c.Key)
			break
		}
		if strings.TrimSpace(c.CertificateFile) != "" || strings.TrimSpace(c.KeyFile) != "" {
			pathOnly = true
		}
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		if pathOnly {
			return nil, []string{fmt.Sprintf("inbound %s: TLS certificate is referenced by file path on the source host; its content is not in the database, so upload the certificate/key on this host after import", row.Tag)}, nil
		}
		return nil, []string{fmt.Sprintf("inbound %s: TLS is enabled but no certificate is present; upload one manually", row.Tag)}, nil
	}
	spec := &tlsCertSpec{
		ServerName:  strings.TrimSpace(t.ServerName),
		Certificate: certPEM,
		KeyPEM:      keyPEM,
		ALPN:        []string(t.ALPN),
		Insecure:    t.AllowInsecure,
	}
	sum := sha256.Sum256([]byte(strings.Join(certPEM, "\n") + "\x00" + strings.Join(keyPEM, "\n")))
	spec.Key = "tls:" + hex.EncodeToString(sum[:])
	spec.Name = "tls-" + firstNonEmpty(spec.ServerName, row.Tag)
	return spec, nil, nil
}

// buildPlainTLSRecord renders a plain-TLS spec into an s-ui TLS record: the
// Server block is a sing-box inbound TLS with the inline certificate/key, the
// Client block is what a subscription link uses to connect.
func buildPlainTLSRecord(spec tlsCertSpec) (model.Tls, error) {
	server := map[string]any{
		"enabled":     true,
		"certificate": spec.Certificate,
		"key":         spec.KeyPEM,
	}
	if spec.ServerName != "" {
		server["server_name"] = spec.ServerName
	}
	if len(spec.ALPN) > 0 {
		server["alpn"] = spec.ALPN
	}
	client := map[string]any{"enabled": true}
	if spec.ServerName != "" {
		client["server_name"] = spec.ServerName
	}
	if spec.Insecure {
		client["insecure"] = true
	}
	if len(spec.ALPN) > 0 {
		client["alpn"] = spec.ALPN
	}
	serverJSON, err := marshalJSON(server)
	if err != nil {
		return model.Tls{}, err
	}
	clientJSON, err := marshalJSON(client)
	if err != nil {
		return model.Tls{}, err
	}
	return model.Tls{Name: spec.Name, Server: serverJSON, Client: clientJSON}, nil
}

// findExistingPlainTLS matches a plain-TLS spec to an existing s-ui TLS record
// by certificate content, so a re-import or scheduled sync reuses it instead of
// creating a duplicate.
func findExistingPlainTLS(tx *gorm.DB, spec tlsCertSpec) (model.Tls, bool, error) {
	var rows []model.Tls
	if err := tx.Model(model.Tls{}).Find(&rows).Error; err != nil {
		return model.Tls{}, false, err
	}
	want := strings.Join(spec.Certificate, "\n")
	for _, row := range rows {
		var server struct {
			Certificate flexStringList `json:"certificate"`
		}
		if err := json.Unmarshal(row.Server, &server); err != nil {
			continue
		}
		if len(server.Certificate) > 0 && strings.Join([]string(server.Certificate), "\n") == want {
			return row, true, nil
		}
	}
	return model.Tls{}, false, nil
}
