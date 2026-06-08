package importxui

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x/database/model"

	"gorm.io/gorm"
)

type xuiStreamSettings struct {
	Network         string            `json:"network"`
	Security        string            `json:"security"`
	TLSSettings     xuiTLSSetting     `json:"tlsSettings"`
	RealitySettings xuiRealitySetting `json:"realitySettings"`
	TCPSettings     map[string]any    `json:"tcpSettings"`
	WSSettings      map[string]any    `json:"wsSettings"`
	GRPCSettings    map[string]any    `json:"grpcSettings"`
	HTTPSettings    map[string]any    `json:"httpSettings"`
	HTTPUPSettings  map[string]any    `json:"httpupgradeSettings"`
}

type xuiRealitySetting struct {
	Target      string              `json:"target"`
	ServerNames []string            `json:"serverNames"`
	PrivateKey  string              `json:"privateKey"`
	MaxTimediff int64               `json:"maxTimediff"`
	ShortIDs    []string            `json:"shortIds"`
	Settings    xuiRealityClientSet `json:"settings"`

	// Outbound (client) reality fields. On an Xray *outbound* the panel stores
	// the peer's public key, single short id, server name and fingerprint at the
	// top level of realitySettings — there is no private key. These tags do not
	// collide with the inbound array/sub-block fields above.
	PublicKey   string `json:"publicKey"`
	ShortID     string `json:"shortId"`
	ServerName  string `json:"serverName"`
	Fingerprint string `json:"fingerprint"`
}

type xuiRealityClientSet struct {
	PublicKey   string `json:"publicKey"`
	Fingerprint string `json:"fingerprint"`
	ServerName  string `json:"serverName"`
}

type realitySpec struct {
	Key         string
	Name        string
	PrivateKey  string
	Target      string
	Host        string
	Port        int
	ServerName  string
	PublicKey   string
	Fingerprint string
	ShortIDs    []string
	MaxTimediff int64
}

func parseStreamSettings(row xuiInboundRow) (xuiStreamSettings, error) {
	var stream xuiStreamSettings
	if len(row.StreamSettings) == 0 {
		return stream, nil
	}
	if err := json.Unmarshal(row.StreamSettings, &stream); err != nil {
		return stream, fmt.Errorf("inbound %s stream_settings: %w", row.Tag, err)
	}
	stream.Network = strings.ToLower(strings.TrimSpace(stream.Network))
	stream.Security = strings.ToLower(strings.TrimSpace(stream.Security))
	return stream, nil
}

func extractReality(row xuiInboundRow) (*realitySpec, []string, error) {
	stream, err := parseStreamSettings(row)
	if err != nil {
		return nil, nil, err
	}
	if stream.Security != "reality" || stream.RealitySettings.PrivateKey == "" {
		return nil, nil, nil
	}

	host, port := splitTarget(stream.RealitySettings.Target)
	serverName := firstNonEmpty(stream.RealitySettings.Settings.ServerName, firstString(stream.RealitySettings.ServerNames), host)
	fingerprint := firstNonEmpty(stream.RealitySettings.Settings.Fingerprint, "chrome")
	spec := &realitySpec{
		PrivateKey:  stream.RealitySettings.PrivateKey,
		Target:      stream.RealitySettings.Target,
		Host:        host,
		Port:        port,
		ServerName:  serverName,
		PublicKey:   stream.RealitySettings.Settings.PublicKey,
		Fingerprint: fingerprint,
		ShortIDs:    stream.RealitySettings.ShortIDs,
		MaxTimediff: stream.RealitySettings.MaxTimediff,
	}
	spec.Key = spec.PrivateKey + "\x00" + spec.Target
	spec.Name = "reality-" + firstNonEmpty(spec.ServerName, spec.Host, row.Tag)

	var warnings []string
	if spec.PublicKey == "" {
		warnings = append(warnings, fmt.Sprintf("inbound %s: reality publicKey is empty; client TLS needs manual review", row.Tag))
	}
	return spec, warnings, nil
}

func splitTarget(target string) (string, int) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", 443
	}
	host, portText, err := net.SplitHostPort(target)
	if err == nil {
		port, _ := strconv.Atoi(portText)
		if port == 0 {
			port = 443
		}
		return strings.Trim(host, "[]"), port
	}
	if i := strings.LastIndex(target, ":"); i > -1 && i < len(target)-1 {
		if port, convErr := strconv.Atoi(target[i+1:]); convErr == nil {
			return strings.Trim(target[:i], "[]"), port
		}
	}
	return target, 443
}

func buildTLSRecord(spec realitySpec) (model.Tls, error) {
	shortIDs := spec.ShortIDs
	if shortIDs == nil {
		shortIDs = []string{}
	}
	server := map[string]any{
		"enabled":     true,
		"server_name": spec.ServerName,
		"reality": map[string]any{
			"enabled": true,
			"handshake": map[string]any{
				"server":      spec.Host,
				"server_port": spec.Port,
			},
			"private_key": spec.PrivateKey,
			"short_id":    shortIDs,
		},
	}
	if spec.MaxTimediff > 0 {
		server["reality"].(map[string]any)["max_time_difference"] = fmt.Sprintf("%dms", spec.MaxTimediff)
	}
	client := map[string]any{
		"enabled":     true,
		"server_name": spec.ServerName,
		"utls": map[string]any{
			"enabled":     true,
			"fingerprint": spec.Fingerprint,
		},
		"reality": map[string]any{
			"enabled":    true,
			"public_key": spec.PublicKey,
			"short_id":   firstString(shortIDs),
		},
	}
	serverJSON, err := marshalJSON(server)
	if err != nil {
		return model.Tls{}, err
	}
	clientJSON, err := marshalJSON(client)
	if err != nil {
		return model.Tls{}, err
	}
	return model.Tls{
		Name:   spec.Name,
		Server: serverJSON,
		Client: clientJSON,
	}, nil
}

func findExistingRealityTLS(tx *gorm.DB, spec realitySpec) (model.Tls, bool, error) {
	var rows []model.Tls
	if err := tx.Model(model.Tls{}).Find(&rows).Error; err != nil {
		return model.Tls{}, false, err
	}
	for _, row := range rows {
		if tlsMatchesReality(row, spec) {
			return row, true, nil
		}
	}
	return model.Tls{}, false, nil
}

func tlsMatchesReality(row model.Tls, spec realitySpec) bool {
	var server struct {
		Reality struct {
			PrivateKey string `json:"private_key"`
			Handshake  struct {
				Server     string `json:"server"`
				ServerPort int    `json:"server_port"`
			} `json:"handshake"`
		} `json:"reality"`
	}
	if err := json.Unmarshal(row.Server, &server); err != nil {
		return false
	}
	targetHost, targetPort := splitTarget(spec.Target)
	return server.Reality.PrivateKey == spec.PrivateKey &&
		server.Reality.Handshake.Server == targetHost &&
		server.Reality.Handshake.ServerPort == targetPort
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
