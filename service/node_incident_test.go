package service

import (
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestNodeIncidentsSurviveRestartWithoutDuplicateTransition(t *testing.T) {
	db := initNodeTestDB(t)
	settings := &SettingService{}
	enableTelegramForTest(t, settings)
	if err := settings.setString("telegramNotifyNodes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := settings.setString("webURI", "https://panel.example/edge/panel/"); err != nil {
		t.Fatal(err)
	}
	sent := make(chan string, 4)
	notifier := newTelegramNotifier(telegramQueueCapacity, func(message string) TelegramResult {
		sent <- message
		return TelegramResult{Success: true}
	}, func(string, map[string]any) {})
	replaceDefaultTelegramNotifierForTest(t, notifier)
	now := time.Now().Truncate(time.Second)
	nodes := &NodeService{peerNow: func() time.Time { return now }}
	if err := nodes.SetMaster(); err != nil {
		t.Fatal(err)
	}
	invite, err := nodes.Invite()
	if err != nil {
		t.Fatal(err)
	}
	peer, err := nodes.RedeemInvite(invite, "incident test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.AcceptPeerSync(peer.UUID, peer.Token, PeerSyncRequest{}); err != nil {
		t.Fatal(err)
	}
	check := func(service *NodeService, count int64, problem string) {
		t.Helper()
		if err := service.ReconcileNodeIncidents(); err != nil {
			t.Fatal(err)
		}
		var total int64
		if err := db.Model(&model.AuditEvent{}).Where("event LIKE ?", "node_incident_%").Count(&total).Error; err != nil || total != count {
			t.Fatalf("expected %d transitions, got %d (%v)", count, total, err)
		}
		view, err := service.Get(peer.UUID)
		if err != nil || view.Incident != problem {
			t.Fatalf("wanted incident %q, got %+v (%v)", problem, view, err)
		}
	}
	check(nodes, 0, "")
	now = now.Add(45 * time.Second)
	check(nodes, 1, "offline")
	offline := receiveString(t, sent, "offline node notification")
	if !strings.Contains(offline, "node_offline") || !strings.Contains(offline, "/edge/panel/nodes?node="+peer.UUID) || !strings.Contains(offline, "incident test") {
		t.Fatal("node notification did not include the event, name and panel link")
	}
	restarted := &NodeService{peerNow: nodes.peerNow}
	check(restarted, 1, "offline")
	if _, err := restarted.AcceptPeerSync(peer.UUID, peer.Token, PeerSyncRequest{}); err != nil {
		t.Fatal(err)
	}
	check(restarted, 2, "")
	if recovered := receiveString(t, sent, "node recovery notification"); !strings.Contains(recovered, "node_recovered") {
		t.Fatal("node recovery notification was not delivered")
	}
	if err := settings.setString("telegramNotifyNodes", "false"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(45 * time.Second)
	check(restarted, 3, "offline")
	select {
	case <-sent:
		t.Fatal("disabled Telegram alerts emitted a node notification")
	case <-time.After(50 * time.Millisecond):
	}
}
