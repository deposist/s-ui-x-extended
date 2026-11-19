package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/sagernet/sing/common/metadata"
)

func TestPeerPublicTransportsReachAcknowledgedSubscription(t *testing.T) {
	tests := []struct {
		name, protocol, transport string
		params                    map[string]string
	}{
		{"WebSocket", "vless", `{"type":"ws","path":"/edge","headers":{"host":["cdn.example"],"Authorization":"transport-secret"}}`, map[string]string{"type": "ws", "path": "/edge", "host": "cdn.example"}},
		{"gRPC", "vmess", `{"type":"grpc","service_name":"api-edge"}`, map[string]string{"net": "grpc", "path": "api-edge"}},
		{"HTTP", "trojan", `{"type":"http","host":["cdn.example","backup.example"],"path":"/edge"}`, map[string]string{"type": "http", "path": "/edge", "host": "cdn.example,backup.example"}},
		{"HTTPUpgrade", "vless", `{"type":"httpupgrade","host":"cdn.example","path":"/edge"}`, map[string]string{"type": "httpupgrade", "path": "/edge", "host": "cdn.example"}},
		{"QUIC", "vmess", `{"type":"quic"}`, map[string]string{"net": "quic"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := model.Inbound{Type: test.protocol, Options: json.RawMessage(`{"listen_port":1443,"transport":` + test.transport + `}`)}
			if test.name == "QUIC" {
				row.TlsId = 1
				row.Tls = &model.Tls{Server: json.RawMessage(`{"enabled":true,"server_name":"slave.example"}`)}
			}
			link := publicPeerLink(row)
			if link == nil {
				t.Fatal("supported transport missing from peer inventory")
			}
			encoded, err := json.Marshal(link)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "transport-secret") {
				t.Fatal("arbitrary header leaked into peer inventory")
			}
			got := peerSubscriptionLink(t, row.Type, link)
			for key, want := range test.params {
				if got[key] != want {
					t.Errorf("%s = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

func TestPeerPublicRealityDerivesKeyAndPublishesClientParameters(t *testing.T) {
	// RFC 7748, section 6.1, Alice's key pair.
	private, err := hex.DecodeString("77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a")
	if err != nil {
		t.Fatal(err)
	}
	public, err := hex.DecodeString("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a")
	if err != nil {
		t.Fatal(err)
	}
	privateKey := base64.RawURLEncoding.EncodeToString(private)
	publicKey := base64.RawURLEncoding.EncodeToString(public)
	row := model.Inbound{
		Type:    "vless",
		TlsId:   1,
		Options: json.RawMessage(`{"listen_port":1443}`),
		Tls: &model.Tls{
			Server: json.RawMessage(`{"enabled":true,"server_name":"cover.example","reality":{"enabled":true,"private_key":"` + privateKey + `","short_id":["abcd1234"],"handshake":{"server":"private-backend.example","server_port":443}}}`),
			Client: json.RawMessage(`{"reality":{"public_key":"obsolete"},"utls":{"enabled":true,"fingerprint":"firefox"}}`),
		},
	}
	link := publicPeerLink(row)
	if link == nil {
		t.Fatal("REALITY missing from peer inventory")
	}
	encoded, err := json.Marshal(link)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{privateKey, "private_key", "private-backend.example", "obsolete"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatal("sensitive or stale TLS configuration leaked into peer inventory")
		}
	}
	got := peerSubscriptionLink(t, row.Type, link)
	for key, want := range map[string]string{"security": "reality", "pbk": publicKey, "sid": "abcd1234", "sni": "cover.example", "fp": "firefox"} {
		if got[key] != want {
			t.Errorf("%s = %q, want %q", key, got[key], want)
		}
	}
}

func TestPeerPublicLinkStopsPublishingAfterLease(t *testing.T) {
	peerSubscriptionLink(t, "vless", &peerPublicLink{Port: 1443})
	db := database.GetDB()
	if err := db.Model(&model.Node{}).Where("enrollment_digest <> ''").
		Update("last_peer_sync", time.Now().Add(-model.PeerAccessLease-time.Second).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	var client model.Client
	if err := db.Where("name = ?", "test-client").Take(&client).Error; err != nil {
		t.Fatal(err)
	}
	if links := ClientNodeLinks(&client); len(links) != 0 {
		t.Fatalf("published %d links for an expired Slave", len(links))
	}
}

func TestPeerPublicTransportsRouteLocalTraffic(t *testing.T) {
	tests := []struct {
		name, protocol, transport, buildTag string
	}{
		{"WebSocket", "vless", `{"type":"ws","path":"/edge","headers":{"Host":"cdn.example"}}`, ""},
		{"gRPC", "vmess", `{"type":"grpc","service_name":"api-edge"}`, ""},
		{"HTTP", "trojan", `{"type":"http","host":"cdn.example","path":"/edge"}`, ""},
		{"HTTPUpgrade", "vless", `{"type":"httpupgrade","host":"cdn.example","path":"/edge"}`, ""},
		{"QUIC", "vmess", `{"type":"quic"}`, "with_quic"},
		{"REALITY", "vless", `null`, "with_utls"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.buildTag != "" && !capabilities.BuildTags()[test.buildTag] {
				t.Skipf("requires %s", test.buildTag)
			}
			db := initNodeTestDB(t)
			ipmonitor.ResetCaches()
			t.Cleanup(ipmonitor.ResetCaches)
			server := core.NewCore()
			t.Cleanup(func() { _ = server.Stop() })
			nodes := &NodeService{Runtime: NewRuntime(server)}
			panel := model.NodePanel{ID: 1, UUID: "peer-test", Role: "slave", PublicHost: "127.0.0.1"}
			if err := db.Create(&panel).Error; err != nil {
				t.Fatal(err)
			}
			network := "anytls" // The existing helper reserves a TCP port for AnyTLS.
			if test.name == "QUIC" {
				network = "vmess"
			}
			port := reservePeerProtocolPort(t, network)
			inbound := model.Inbound{
				Type: test.protocol, Tag: "peer-transport",
				Options: json.RawMessage(fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":%d,"transport":%s}`, port, test.transport)),
				OutJson: json.RawMessage(`{}`), Addrs: json.RawMessage(`[]`),
			}
			certificate := testPeerTransportTLS(t, &inbound, test.name)
			if inbound.Tls != nil {
				if err := db.Create(inbound.Tls).Error; err != nil {
					t.Fatal(err)
				}
				inbound.TlsId = inbound.Tls.Id
			}
			if err := db.Create(&inbound).Error; err != nil {
				t.Fatal(err)
			}
			link := publicPeerLink(inbound)
			if link == nil {
				t.Fatal("local inbound did not announce a share link")
			}
			user := peerUser{InboundID: inbound.Id, ClientID: 1, Type: test.protocol, Name: "alice"}
			if test.protocol == "trojan" {
				user.Password = "test-password"
			} else {
				user.UUID = "c46a4d36-c8f2-4aa7-828a-8df136220995"
			}
			state := PeerSyncResponse{Users: []peerUser{user}}
			bindLocalPeerInventory(t, nodes, panel, &state)
			if err := server.Start([]byte(`{}`)); err != nil {
				t.Fatal(err)
			}
			if err := nodes.applyPeerState(panel, state); err != nil {
				t.Fatal(err)
			}
			client := startPeerProtocolClient(t, link.subscriptionInbound(PeerInbound{Type: inbound.Type, Tag: inbound.Tag}), user, panel.PublicHost, certificate)
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(target.Close)
			transport := &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return client.DialContext(ctx, network, metadata.ParseSocksaddr(address))
			}}
			t.Cleanup(transport.CloseIdleConnections)
			resp, err := (&http.Client{Transport: transport, Timeout: 5 * time.Second}).Get(target.URL)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				t.Errorf("expected loopback target status 204, got %d", resp.StatusCode)
			}
		})
	}
}

func testPeerTransportTLS(t *testing.T, inbound *model.Inbound, name string) string {
	t.Helper()
	var server map[string]any
	var certificate string
	switch name {
	case "QUIC":
		var key string
		certificate, key = peerProtocolTestPEM(t)
		server = map[string]any{"enabled": true, "server_name": "example.com", "certificate": certificate, "key": key}
	case "REALITY":
		cover := httptest.NewUnstartedServer(nil)
		cover.EnableHTTP2 = true
		cover.StartTLS()
		t.Cleanup(cover.Close)
		coverPort := cover.Listener.Addr().(*net.TCPAddr).Port
		server = map[string]any{
			"enabled": true, "server_name": "example.com",
			"reality": map[string]any{
				"enabled": true, "private_key": "dwdtCnMYpX08FsFyUbJmRd9ML4frwJkqsXf7pR25LCo",
				"short_id":  []string{"abcd1234"},
				"handshake": map[string]any{"server": "127.0.0.1", "server_port": coverPort},
			},
		}
	default:
		return ""
	}
	data, err := json.Marshal(server)
	if err != nil {
		t.Fatal(err)
	}
	inbound.Tls = &model.Tls{Name: "transport-test", Server: data, Client: json.RawMessage(`{}`)}
	return certificate
}

func peerSubscriptionLink(t *testing.T, protocol string, link *peerPublicLink) map[string]string {
	t.Helper()
	db := initNodeTestDB(t)
	nodes := &NodeService{}
	if err := nodes.SetMaster(); err != nil {
		t.Fatal(err)
	}
	code, err := nodes.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := nodes.RedeemInvite(code, "slave")
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "test-client")
	client.Config = json.RawMessage(`{"vless":{"uuid":"c46a4d36-c8f2-4aa7-828a-8df136220995"},"trojan":{"password":"test-password"},"vmess":{"uuid":"c46a4d36-c8f2-4aa7-828a-8df136220995"}}`)
	if err := db.Save(&client).Error; err != nil {
		t.Fatal(err)
	}
	input := PeerSyncRequest{PublicHost: "slave.example", Inbounds: []PeerInbound{{ID: 23, Tag: "peer", Type: protocol, Link: link}}}
	sync := func() PeerSyncResponse {
		t.Helper()
		out, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, input)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	sync()
	if _, err := nodes.ChangePeerClients(peer.UUID, 23, []uint{client.Id}, true); err != nil {
		t.Fatal(err)
	}
	input.AppliedRevision = sync().Revision
	sync()
	links := ClientNodeLinks(&client)
	if len(links) != 1 {
		t.Fatalf("expected one acknowledged subscription link, got %d", len(links))
	}
	uri := links[0]["uri"]
	params := make(map[string]string)
	if protocol == "vmess" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, "vmess://"))
		if err != nil {
			t.Fatal(err)
		}
		var values map[string]any
		if err := json.Unmarshal(data, &values); err != nil {
			t.Fatal(err)
		}
		for key, value := range values {
			params[key], _ = value.(string)
		}
		return params
	}
	u, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	for key, values := range u.Query() {
		params[key] = values[0]
	}
	return params
}
