package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/core/capabilities"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/deposist/s-ui-x-extended/util"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/metadata"
)

func TestPeerAdditionalProtocolsRouteAndEnforceQuota(t *testing.T) {
	for _, kind := range []string{"anytls", "hysteria", "hysteria2", "tuic"} {
		t.Run(kind, func(t *testing.T) {
			if kind != "anytls" && !capabilities.BuildTags()["with_quic"] {
				t.Skip("requires with_quic")
			}
			testPeerProtocolDataPlane(t, kind)
		})
	}
}

func testPeerProtocolDataPlane(t *testing.T, kind string) {
	t.Helper()
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
	port := reservePeerProtocolPort(t, kind)
	certificate, key := peerProtocolTestPEM(t)
	serverTLS, err := json.Marshal(map[string]any{"enabled": true, "server_name": "example.com", "certificate": certificate, "key": key})
	if err != nil {
		t.Fatal(err)
	}
	tls := model.Tls{Name: "test", Server: serverTLS, Client: json.RawMessage(`{}`)}
	if err := db.Create(&tls).Error; err != nil {
		t.Fatal(err)
	}
	options := fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":%d}`, port)
	if kind == "hysteria" {
		options = fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":%d,"up_mbps":50,"down_mbps":50}`, port)
	}
	inbound := model.Inbound{Type: kind, Tag: "peer-test-" + kind, TlsId: tls.Id, Tls: &tls, Options: json.RawMessage(options), OutJson: json.RawMessage(`{}`), Addrs: json.RawMessage(`[]`)}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	link := publicPeerLink(inbound)
	if link == nil {
		t.Fatal("eligible inbound did not advertise a subscription link")
	}
	user := peerUser{InboundID: inbound.Id, ClientID: 1, Type: kind, Name: "alice", Volume: 32768, LimitIP: 1, IPLimitMode: ipmonitor.ModeEnforce}
	switch kind {
	case "hysteria":
		user.AuthString = "test-auth+ &:@"
	case "tuic":
		user.UUID = "c46a4d36-c8f2-4aa7-828a-8df136220995"
		user.Password = "test-pass+ &:@"
	default:
		user.Password = "test-pass+ &:@"
	}
	state := PeerSyncResponse{Users: []peerUser{user}}
	if err := server.Start([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	apply := func() {
		t.Helper()
		bindLocalPeerInventory(t, nodes, panel, &state)
		if err := nodes.applyPeerState(panel, state); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	subscription := link.subscriptionInbound(PeerInbound{ID: inbound.Id, Tag: inbound.Tag, Type: kind})
	client := startPeerProtocolClient(t, subscription, user, panel.PublicHost, certificate)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(endpoint.Close)
	destination := metadata.ParseSocksaddr(endpoint.Listener.Addr().String())
	request := func() error {
		transport := &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return client.DialContext(ctx, network, destination)
		}}
		defer transport.CloseIdleConnections()
		resp, err := (&http.Client{Transport: transport, Timeout: 4 * time.Second}).Get(endpoint.URL)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("unexpected test server status %d", resp.StatusCode)
		}
		return nil
	}
	if err := request(); err != nil {
		t.Fatalf("subscription credential did not route TCP: %v", err)
	}
	if ipmonitor.ObserveAndAllow("alice", "192.0.2.1") {
		t.Fatal("TCP route did not enforce the assigned user's IP limit")
	}
	var usage model.PeerTraffic
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, user.ClientID).Error; err != nil || usage.Used <= 0 {
		t.Fatalf("TCP traffic did not debit the assigned user's quota (%d bytes): %v", usage.Used, err)
	}
	exchange := peerProtocolUDPExchange(t, client)
	const payload = "paired protocol UDP quota probe"
	beforeUDP := usage.Used
	if echo, err := exchange(payload); err != nil || echo != payload {
		t.Fatalf("subscription credential did not route UDP: %v, reply %q", err, echo)
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, user.ClientID).Error; err != nil || usage.Used-beforeUDP != int64(2*len(payload)) {
		t.Fatalf("UDP usage delta %d, want %d: %v", usage.Used-beforeUDP, 2*len(payload), err)
	}
	state.Users[0].Volume = usage.Used
	apply()
	client = startPeerProtocolClient(t, subscription, user, panel.PublicHost, certificate)
	if err := request(); err == nil {
		t.Fatal("exhausted user continued to route TCP")
	}
	exchange = peerProtocolUDPExchange(t, client)
	if _, err := exchange(payload); err == nil {
		t.Fatal("exhausted user continued to route UDP")
	}
	state.Users = []peerUser{}
	apply()
	client = startPeerProtocolClient(t, subscription, user, panel.PublicHost, certificate)
	if err := request(); err == nil {
		t.Fatal("revoked user continued to route TCP")
	}
}

func reservePeerProtocolPort(t *testing.T, kind string) int {
	t.Helper()
	if kind == "anytls" {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		return listener.Addr().(*net.TCPAddr).Port
	}
	listener, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.LocalAddr().(*net.UDPAddr).Port
}

func peerProtocolTestPEM(t *testing.T) (string, string) {
	t.Helper()
	endpoint := httptest.NewTLSServer(nil)
	defer endpoint.Close()
	cert := endpoint.TLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
}

func startPeerProtocolClient(t *testing.T, inbound model.Inbound, user peerUser, host, certificate string) adapter.Outbound {
	t.Helper()
	credential, err := json.Marshal(map[string]peerUser{user.Type: user})
	if err != nil {
		t.Fatal(err)
	}
	links := util.LinkGenerator(credential, &inbound, host)
	if len(links) != 1 {
		t.Fatalf("expected a single subscription URI, got %d", len(links))
	}
	parsed, tag, err := util.GetOutbound(links[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	if certificate != "" {
		options, ok := (*parsed)["tls"].(map[string]any)
		if !ok {
			t.Fatal("subscription omitted TLS options")
		}
		options["certificate"] = certificate
	}
	config, err := json.Marshal(map[string]any{"outbounds": []map[string]any{*parsed}})
	if err != nil {
		t.Fatal(err)
	}
	client := core.NewCore()
	t.Cleanup(func() { _ = client.Stop() })
	if err := client.Start(config); err != nil {
		t.Fatal(err)
	}
	outbound, ok := client.GetInstance().Outbound().Outbound(tag)
	if !ok {
		t.Fatal("subscription outbound did not start")
	}
	return outbound
}

func peerProtocolUDPExchange(t *testing.T, client adapter.Outbound) func(string) (string, error) {
	t.Helper()
	echo, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = echo.Close() })
	go func() {
		buffer := make([]byte, 512)
		for {
			n, from, err := echo.ReadFrom(buffer)
			if err != nil {
				return
			}
			if _, err := echo.WriteTo(buffer[:n], from); err != nil {
				return
			}
		}
	}()
	destination := metadata.ParseSocksaddr(echo.LocalAddr().String())
	return func(payload string) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		conn, err := client.ListenPacket(ctx, destination)
		if err != nil {
			return "", err
		}
		defer conn.Close()
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return "", fmt.Errorf("set UDP read deadline: %w", err)
		}
		if _, err := conn.WriteTo([]byte(payload), destination); err != nil {
			return "", fmt.Errorf("send UDP packet: %w", err)
		}
		buffer := make([]byte, 512)
		n, _, err := conn.ReadFrom(buffer)
		return string(buffer[:n]), err
	}
}
