package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/ipmonitor"
	"github.com/sagernet/sing-vmess/vless"
	"github.com/sagernet/sing/common/metadata"
	"golang.org/x/net/dns/dnsmessage"
)

func bindLocalPeerInventory(t *testing.T, nodes *NodeService, panel model.NodePanel, state *PeerSyncResponse) {
	t.Helper()
	inbounds, err := announcePeerInbounds(nodes.db())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodePeerInventory(inbounds)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.bindInventory(encoded, panel.PublicHost); err != nil {
		t.Fatal(err)
	}
}

func TestPeerAccessRequiresRunningCoreAndPersistsSeparately(t *testing.T) {
	db := initNodeTestDB(t)
	instance := core.NewCore()
	t.Cleanup(func() { _ = instance.Stop() })
	runtime := NewRuntime(instance)
	nodes := &NodeService{Runtime: runtime}
	panel := model.NodePanel{ID: 1, UUID: "peer", Role: "slave"}
	if err := db.Create(&panel).Error; err != nil {
		t.Fatal(err)
	}
	state := PeerSyncResponse{Users: []peerUser{}}
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err == nil {
		t.Fatal("applied a revision with a stopped core")
	}
	if err := instance.Start([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	stale := PeerSyncResponse{Users: []peerUser{}}
	if err := stale.bindInventory([]byte(`{}`), "old.example"); err != nil {
		t.Fatal(err)
	}
	if err := nodes.applyPeerState(panel, stale); !errors.Is(err, ErrInvalidPeerInventory) {
		t.Fatalf("expected stale inventory rejection, got %v", err)
	}
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	stored, err := nodes.Panel()
	if err != nil || !instance.IsRunning() || stored.AppliedRevision != state.Revision {
		t.Fatalf("revision was not persisted on the active panel: %v", err)
	}
	if err := nodes.UnlinkSlave(); err != nil {
		t.Fatal(err)
	}
	stored, err = nodes.Panel()
	if err != nil || stored.Role != "standalone" || stored.AppliedRevision != "" {
		t.Fatalf("unlink retained applied state: %v", err)
	}
}

func TestPeerVLESSDataPlaneApplyAndUnlink(t *testing.T) {
	db := initNodeTestDB(t)
	instance := core.NewCore()
	t.Cleanup(func() { _ = instance.Stop() })
	nodes := &NodeService{Runtime: NewRuntime(instance)}
	panel := model.NodePanel{ID: 1, UUID: "peer", Role: "slave"}
	if err := db.Create(&panel).Error; err != nil {
		t.Fatal(err)
	}
	reserve := func() net.Listener {
		t.Helper()
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		return listener
	}
	slaveListener := reserve()
	slavePort := slaveListener.Addr().(*net.TCPAddr).Port
	_ = slaveListener.Close()
	inbound := model.Inbound{Type: "vless", Tag: "peer-vless", Options: json.RawMessage(fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":%d}`, slavePort))}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	const credential = "c46a4d36-c8f2-4aa7-828a-8df136220995"
	ipmonitor.ResetCaches()
	t.Cleanup(ipmonitor.ResetCaches)
	state := PeerSyncResponse{Users: []peerUser{{InboundID: inbound.Id, ClientID: 1, Type: "vless", Name: "alice", UUID: credential, LimitIP: 1, IPLimitMode: ipmonitor.ModeEnforce}}}
	bindLocalPeerInventory(t, nodes, panel, &state)
	settings := &SettingService{}
	if err := settings.SetConfig(`{
		"dns":{"servers":[{"type":"hosts","tag":"test-hosts","predefined":{"quota.example.":["192.0.2.42"]}}]},
		"route":{"rules":[{"inbound":["peer-vless"],"port":[53],"action":"hijack-dns"}]}
	}`); err != nil {
		t.Fatal(err)
	}
	if err := instance.Start([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	client, err := vless.NewClient(credential, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	held := make(chan struct{}, 1)
	endpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/hold" {
			writer.Header().Set("Content-Length", "1")
			writer.WriteHeader(http.StatusOK)
			writer.(http.Flusher).Flush()
			held <- struct{}{}
			<-request.Context().Done()
			return
		}
		if request.URL.Path == "/payload" {
			_, _ = writer.Write(make([]byte, 8192))
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer endpoint.Close()
	connect := func(path string) error {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", slavePort), time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(7 * time.Second)); err != nil {
			return err
		}
		request, err := http.NewRequest(http.MethodGet, endpoint.URL+path, nil)
		if err != nil {
			return err
		}
		stream, err := client.DialConn(conn, metadata.ParseSocksaddr(request.URL.Host))
		if err != nil {
			return err
		}
		if err := request.Write(stream); err != nil {
			return err
		}
		response, err := http.ReadResponse(bufio.NewReader(stream), request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if path == "/payload" && response.StatusCode == http.StatusOK {
			length, err := io.Copy(io.Discard, response.Body)
			if err != nil || length != 8192 {
				return fmt.Errorf("incomplete payload (%d bytes): %w", length, err)
			}
			return nil
		}
		if path == "/hold" && response.StatusCode == http.StatusOK && response.ContentLength == 1 {
			_, err := io.ReadAll(response.Body)
			if errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return fmt.Errorf("open stream was not closed by Slave: %v", err)
		}
		if response.StatusCode != http.StatusNoContent {
			return fmt.Errorf("unexpected test server status: %d", response.StatusCode)
		}
		return nil
	}
	if err := connect(""); err != nil {
		t.Fatalf("assigned user could not reach its target through Slave: %v", err)
	}
	if ipmonitor.ObserveAndAllow("alice", "192.0.2.1") {
		t.Fatal("Slave ignored the per-client IP limit")
	}
	expires := time.Now().Add(4 * time.Second).Unix()
	state.Users[0].Expiry = expires
	state.Users[0].Connections = 1
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- connect("/hold") }()
	select {
	case <-held:
	case err := <-finished:
		t.Fatalf("failed to hold first connection: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("first connection never reached target")
	}
	if err := connect(""); err == nil || time.Now().Unix() >= expires {
		t.Fatal("Slave did not reject a concurrent connection before expiry")
	}
	if err := <-finished; err != nil || time.Now().Unix() < expires {
		t.Fatalf("Slave did not close the established stream at expiry while offline: %v", err)
	}
	if err := connect(""); err == nil {
		t.Fatal("Slave accepted a new connection after expiry")
	}
	state.Users[0].Expiry = 0
	state.Users[0].Download = 4096
	state.Users[0].Volume = 16384
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := connect("/payload"); err != nil || time.Since(started) < 900*time.Millisecond {
		t.Fatalf("download did not complete at the configured rate: %v (%v)", err, time.Since(started))
	}
	var usage model.PeerTraffic
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.Used < 8192 {
		t.Fatalf("traffic was not persisted: %v (%d bytes)", err, usage.Used)
	}
	state.Users[0].Volume = usage.Used
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err == nil {
		t.Fatal("Slave ignored consumed quota after restarting the proxy")
	}
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	go func() {
		buffer := make([]byte, 512)
		for {
			length, address, err := udp.ReadFrom(buffer)
			if err != nil {
				return
			}
			if _, err := udp.WriteTo(buffer[:length], address); err != nil {
				return
			}
		}
	}()
	const packet = "slave UDP quota probe"
	exchange := func(destination metadata.Socksaddr, payload []byte) ([]byte, error) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", slavePort), time.Second)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return nil, err
		}
		stream, err := client.DialEarlyPacketConn(conn, destination)
		if err != nil {
			return nil, err
		}
		if _, err := stream.WriteTo(payload, destination); err != nil {
			return nil, err
		}
		buffer := make([]byte, 512)
		length, _, err := stream.ReadFrom(buffer)
		return buffer[:length], err
	}
	echo := metadata.ParseSocksaddr(udp.LocalAddr().String())
	state.Users[0].Volume += 2 * int64(len(packet))
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if reply, err := exchange(echo, []byte(packet)); err != nil || string(reply) != packet {
		t.Fatalf("first VLESS UDP exchange failed: %v, reply %q", err, reply)
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.Used != state.Users[0].Volume {
		t.Fatalf("UDP upload and download did not consume the remaining quota: %v (%d bytes)", err, usage.Used)
	}
	if _, err := exchange(echo, []byte(packet)); err == nil {
		t.Fatal("Slave allowed another UDP exchange after quota exhaustion")
	}
	dnsDestination := metadata.ParseSocksaddr("127.0.0.1:53")
	query := dnsmessage.Message{
		Header: dnsmessage.Header{ID: 21, RecursionDesired: true},
		Questions: []dnsmessage.Question{{
			Name: dnsmessage.MustNewName("quota.example."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET,
		}},
	}
	request, err := query.Pack()
	if err != nil {
		t.Fatal(err)
	}
	beforeDNS := usage.Used
	state.Users[0].Volume += 512
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	reply, err := exchange(dnsDestination, request)
	if err != nil {
		t.Fatalf("built-in DNS did not answer through Slave: %v", err)
	}
	var response dnsmessage.Message
	if err := response.Unpack(reply); err != nil {
		t.Fatal(err)
	}
	if !response.Response || response.ID != query.ID || response.RCode != dnsmessage.RCodeSuccess || len(response.Answers) != 1 {
		t.Fatalf("unexpected built-in DNS reply: %+v", response)
	}
	if answer, ok := response.Answers[0].Body.(*dnsmessage.AResource); !ok || answer.A != [4]byte{192, 0, 2, 42} {
		t.Fatalf("unexpected built-in DNS answer: %+v", response.Answers[0])
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.Used != beforeDNS+int64(len(request)+len(reply)) {
		t.Fatalf("built-in DNS traffic was not fully accounted for: %v (%d bytes)", err, usage.Used-beforeDNS)
	}
	state.Users[0].Volume = usage.Used
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if _, err := exchange(dnsDestination, request); err == nil {
		t.Fatal("Slave allowed a DNS exchange after quota exhaustion")
	}
	state.Users[0].Volume = 0
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&panel).UpdateColumn("last_sync", time.Now().Add(-time.Hour).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err == nil {
		t.Fatal("Slave accepted a credential after its last confirmation expired")
	}
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err != nil {
		t.Fatalf("a fresh confirmation of the same revision did not restore access: %v", err)
	}
	leaseEnds := time.Now().Add(3 * time.Second).Unix()
	if err := db.Model(&panel).UpdateColumn("last_sync", leaseEnds-int64(model.PeerAccessLease/time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	go func() { finished <- connect("/hold") }()
	select {
	case <-held:
	case err := <-finished:
		t.Fatalf("access was denied during the offline grace period: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("offline connection never reached target")
	}
	if err := <-finished; err != nil || time.Now().Unix() < leaseEnds {
		t.Fatalf("the offline grace period did not close the established stream: %v", err)
	}
	if err := instance.Stop(); err != nil {
		t.Fatal(err)
	}
	config := NewConfigServiceWithRuntime(nodes.runtime())
	proxyConfig, err := config.getConfig(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.Start(*proxyConfig); err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err == nil {
		t.Fatal("restarting the Slave renewed an expired confirmation")
	}

	state.Users[0].AutoReset, state.Users[0].Period, state.Users[0].Volume = true, 86400, usage.Used
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err == nil {
		t.Fatal("Slave reset the quota before the local reset time")
	}
	if err := db.Model(&usage).UpdateColumn("reset_at", time.Now().Add(-time.Minute).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err != nil {
		t.Fatalf("Slave did not reset its quota without consulting Master: %v", err)
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.Used >= state.Users[0].Volume || usage.ResetAt <= time.Now().Unix() {
		t.Fatalf("Slave did not persist its next quota window: %v (%+v)", err, usage)
	}

	state.Users[0].DelayStart, state.Users[0].AutoReset, state.Users[0].Period, state.Users[0].Volume = true, false, 4, 0
	bindLocalPeerInventory(t, nodes, panel, &state)
	if err := nodes.applyPeerState(panel, state); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.StartedAt != 0 {
		t.Fatalf("applying the revision activated the client before local traffic: %v", err)
	}
	go func() { finished <- connect("/hold") }()
	select {
	case <-held:
	case err := <-finished:
		t.Fatalf("first local use failed: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("first local connection never reached target")
	}
	if err := db.First(&usage, "node_uuid = ? AND client_id = ?", panel.UUID, 1).Error; err != nil || usage.StartedAt == 0 {
		t.Fatalf("first traffic did not activate the local timer: %v", err)
	}
	if err := <-finished; err != nil || time.Now().Unix() < usage.StartedAt+state.Users[0].Period {
		t.Fatalf("the delayed expiry did not close the established stream: %v", err)
	}
	if err := connect(""); err == nil {
		t.Fatal("Slave allowed a fresh connection after the delayed expiry")
	}
	if err := nodes.UnlinkSlave(); err != nil {
		t.Fatal(err)
	}
	if err := connect(""); err == nil {
		t.Fatal("user still connected through Slave after unlink")
	}
}

func TestPeerDesiredAccessIsScopedAndFailsClosed(t *testing.T) {
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
	client := nodeTestClient(t, db, "alice")
	unassigned := nodeTestClient(t, db, "unassigned")
	client.Config = json.RawMessage(`{"vless":{"name":"alice","uuid":"c46a4d36-c8f2-4aa7-828a-8df136220995","unused":"not-on-wire"}}`)
	client.Expiry = time.Now().Add(time.Hour).Unix()
	if err := db.Save(&client).Error; err != nil {
		t.Fatal(err)
	}
	input := PeerSyncRequest{PublicHost: "slave.example", Inbounds: []PeerInbound{{ID: 23, Tag: "peer-vless", Type: "vless", Link: &peerPublicLink{Port: 1443}}}}
	sync := func() PeerSyncResponse {
		t.Helper()
		response, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, input)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	initial := sync()
	if _, err := nodes.ChangePeerClients(peer.UUID, 23, []uint{client.Id}, true); err != nil {
		t.Fatal(err)
	}
	assigned := sync()
	if len(assigned.Users) != 1 || assigned.Users[0].Name != "alice" || assigned.Users[0].Expiry != client.Expiry || assigned.Revision == initial.Revision || assigned.Revision != sync().Revision {
		t.Fatal("expected a stable revision for exactly the assigned client")
	}
	if len(ClientNodeLinks(&client)) != 0 {
		t.Fatal("published access before acknowledgement")
	}
	if err := db.Model(&unassigned).UpdateColumn("config", client.Config).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.ChangePeerClients(peer.UUID, 23, []uint{unassigned.Id}, true); err != nil {
		t.Fatal(err)
	}
	if collision := sync(); len(collision.Users) != 0 || collision.Blocked != 2 {
		t.Fatal("a duplicate authentication identity on the same inbound retained access")
	}
	if _, err := nodes.ChangePeerClients(peer.UUID, 23, []uint{unassigned.Id}, false); err != nil {
		t.Fatal(err)
	}
	input.AppliedRevision = assigned.Revision
	sync()
	if links := ClientNodeLinks(&client); len(links) != 1 {
		t.Fatal("acknowledged access must publish without a local inbound")
	}
	if err := db.Create(&model.ClientLimits{ClientId: client.Id, Connections: 1, DownloadSpeed: 8192, UploadSpeed: 4096}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&client).Update("volume", 1024).Error; err != nil {
		t.Fatal(err)
	}
	if len(ClientNodeLinks(&client)) != 0 {
		t.Fatal("published a link after the policy changed and before the next sync")
	}
	limited := sync()
	if len(limited.Users) != 1 || limited.Blocked != 0 || limited.Revision == assigned.Revision {
		t.Fatal("connection and speed limits must be applied in a new revision")
	}
	if user := limited.Users[0]; user.Connections != 1 || user.Download != 8192 || user.Upload != 4096 || user.Speed != 0 || user.Volume != 1024 {
		t.Fatal("invalid split policy")
	}
	input.LeaseCapable = true
	for _, autoReset := range []bool{false, true} {
		if err := db.Model(&client).Updates(map[string]any{"delay_start": true, "auto_reset": autoReset, "reset_days": 2}).Error; err != nil {
			t.Fatal(err)
		}
		if scheduled := sync(); len(scheduled.Users) != 1 || scheduled.Blocked != 0 || scheduled.Revision == limited.Revision {
			t.Fatalf("Slave did not receive the client schedule (auto reset %t)", autoReset)
		}
	}
	if err := db.Model(&client).UpdateColumn("up", 1024).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := (&ClientService{}).DepleteClients(); err != nil {
		t.Fatal(err)
	}
	if scheduled := sync(); len(scheduled.Users) != 1 || scheduled.Blocked != 0 || !scheduled.Users[0].DelayStart {
		t.Fatal("local usage on Master changed the independent Slave schedule")
	}
	if err := db.Model(&client).Updates(map[string]any{"enable": false, "next_reset": 1}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := (&ClientService{}).DepleteClients(); err != nil {
		t.Fatal(err)
	}
	if disabled := sync(); len(disabled.Users) != 0 {
		t.Fatal("a scheduled reset reenabled an explicitly disabled assigned client")
	}
	if err := nodes.RequestPeerDisconnect(peer.UUID); err != nil {
		t.Fatal(err)
	}
	drain := sync()
	if len(drain.Users) != 0 || drain.Detached {
		t.Fatal("must wait for Slave to acknowledge cleanup")
	}
	input.AppliedRevision = drain.Revision
	if !sync().Detached {
		t.Fatal("cleanup was not acknowledged")
	}
	input.PublicHost = "new-slave.example"
	if retry := sync(); !retry.Detached || retry.Revision != drain.Revision || len(retry.Users) != 0 {
		t.Fatal("cleanup receipt could not be recovered after losing the response")
	}
	input.AppliedRevision = ""
	if _, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, input); !errors.Is(err, ErrInvalidNodeCredential) {
		t.Fatalf("credential accepted after disconnect: %v", err)
	}
}

func TestNodeSyncRejectsCredentialsFromAnotherNode(t *testing.T) {
	service := &NodeService{}
	initNodeTestDB(t)
	if err := service.SetMaster(); err != nil {
		t.Fatal(err)
	}
	enroll := func(name string) nodeEnrollment {
		t.Helper()
		code, err := service.Invite()
		if err != nil {
			t.Fatal(err)
		}
		credential, err := service.RedeemInvite(code, name)
		if err != nil {
			t.Fatal(err)
		}
		return credential
	}
	first, second := enroll("first"), enroll("second")
	if _, err := service.AcceptPeerSync(first.UUID, second.Token, PeerSyncRequest{}); !errors.Is(err, ErrInvalidNodeCredential) {
		t.Fatalf("cross-node credential was not rejected: %v", err)
	}
	if _, err := service.AcceptPeerSync(first.UUID, first.Token, PeerSyncRequest{PublicHost: "https://invalid.example"}); !errors.Is(err, ErrInvalidPeerInventory) {
		t.Fatalf("invalid host was not rejected: %v", err)
	}
	if _, err := service.AcceptPeerSync(first.UUID, first.Token, PeerSyncRequest{PublicHost: "slave.example", Inbounds: []PeerInbound{{ID: 23, Tag: "local-vless", Type: "vless"}}}); err != nil {
		t.Fatal(err)
	}
	firstNode, err := enrolledNode(service.db(), first.UUID, first.Token)
	if err != nil {
		t.Fatal(err)
	}
	if inbounds := DecodeNodeInventory(firstNode.Inventory).Inbounds; firstNode.LastPeerSync == 0 || firstNode.LinkHost != "slave.example" || len(inbounds) != 1 || inbounds[0].ID != 23 {
		t.Fatal("authenticated inventory or heartbeat not stored")
	}
	secondNode, err := enrolledNode(service.db(), second.UUID, second.Token)
	if err != nil || secondNode.LastPeerSync != 0 || len(secondNode.Inventory) != 0 {
		t.Fatalf("sync changed a different node: %v", err)
	}
	if err := service.Delete(first.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AcceptPeerSync(first.UUID, first.Token, PeerSyncRequest{}); !errors.Is(err, ErrInvalidNodeCredential) {
		t.Fatalf("force-disconnected node renewed its confirmation: %v", err)
	}
}

func TestPairedAssignmentsFollowInboundIdentity(t *testing.T) {
	db := initNodeTestDB(t)
	service := &NodeService{}
	if err := service.SetMaster(); err != nil {
		t.Fatal(err)
	}
	code, err := service.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := service.RedeemInvite(code, "slave")
	if err != nil {
		t.Fatal(err)
	}
	client := nodeTestClient(t, db, "alice")
	announce := func(inboundID uint, tag string) {
		t.Helper()
		if _, err := service.AcceptPeerSync(peer.UUID, peer.Token, PeerSyncRequest{Inbounds: []PeerInbound{{ID: inboundID, Tag: tag, Type: "vless"}}}); err != nil {
			t.Fatal(err)
		}
	}
	assertAssigned := func(inboundID uint, want int64) {
		t.Helper()
		page, err := service.PeerClients(peer.UUID, inboundID, 25, 0, "", true)
		if err != nil || page.Total != want {
			t.Fatalf("inbound %d: got %+v (%v), want %d assigned", inboundID, page, err, want)
		}
	}
	announce(23, "original")
	if _, err := service.ChangePeerClients(peer.UUID, 24, []uint{client.Id}, true); err == nil {
		t.Fatal("accepted an inbound not present on the selected peer")
	}
	if _, err := service.ChangePeerClients(peer.UUID, 23, []uint{client.Id, client.Id}, true); err != nil {
		t.Fatal(err)
	}
	announce(23, "renamed")
	assertAssigned(23, 1)
	announce(24, "renamed")
	assertAssigned(24, 0)
	announce(23, "renamed")
	assertAssigned(23, 0)
}
