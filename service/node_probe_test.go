package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeProbeChecksHTTPSViaScopedVLESS(t *testing.T) {
	db := initNodeTestDB(t)
	slaveCore := core.NewCore()
	t.Cleanup(func() { _ = slaveCore.Stop() })
	nodes := &NodeService{Runtime: NewRuntime(slaveCore)}
	panel := model.NodePanel{ID: 1, UUID: "test-slave", Role: "slave", PublicHost: "127.0.0.1"}
	if err := db.Create(&panel).Error; err != nil {
		t.Fatal(err)
	}
	port := reservePeerProtocolPort(t, "vless")
	inbound := model.Inbound{Type: "vless", Tag: "public", OutJson: json.RawMessage(`{}`), Addrs: json.RawMessage(`[]`),
		Options: json.RawMessage(fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":%d}`, port))}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := slaveCore.Start([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	control, err := url.Parse(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	dest := netip.MustParseAddrPort(control.Host)
	roots := x509.NewCertPool()
	roots.AddCert(target.Certificate())
	peer := PeerInbound{ID: inbound.Id, Tag: inbound.Tag, Type: inbound.Type, Link: publicPeerLink(inbound)}
	access, err := newPeerProbeAccess(peer, dest, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	state := PeerSyncResponse{Users: []peerUser{}}
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	instance := slaveCore.GetInstance()
	state.Probe = &access
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if slaveCore.GetInstance() != instance {
		t.Fatal("starting a probe restarted the entire Slave proxy")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	if stage, result, _ := runPeerHTTPSProbe(ctx, peer, "127.0.0.1", netip.MustParseAddr("127.0.0.1"), control, dest, access, roots); stage != "https" || result != "success" {
		t.Fatalf("HTTPS check through authorized inbound failed: %s/%s", stage, result)
	}
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer other.Close()
	otherURL, err := url.Parse(other.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, result, _ := runPeerHTTPSProbe(ctx, peer, "127.0.0.1", netip.MustParseAddr("127.0.0.1"), otherURL, netip.MustParseAddrPort(otherURL.Host), access, roots); result == "success" {
		t.Fatal("probe credential could reach a different destination")
	}
	state.Probe = nil
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if slaveCore.GetInstance() != instance {
		t.Fatal("removing a probe restarted the entire Slave proxy")
	}
	if _, result, _ := runPeerHTTPSProbe(ctx, peer, "127.0.0.1", netip.MustParseAddr("127.0.0.1"), control, dest, access, roots); result == "success" {
		t.Fatal("probe credential stayed active after cleanup")
	}
}
