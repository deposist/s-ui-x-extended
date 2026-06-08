package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/realtime"
	"gorm.io/gorm"
)

func TestTrafficDeltasAggregatesByResourceAndTag(t *testing.T) {
	deltas := trafficDeltas([]model.Stats{
		{Resource: "user", Tag: "alice", Direction: true, Traffic: 10},
		{Resource: "user", Tag: "alice", Direction: false, Traffic: 3},
		{Resource: "user", Tag: "alice", Direction: true, Traffic: 7},
		{Resource: "inbound", Tag: "mixed-in", Direction: false, Traffic: 4},
	})

	if len(deltas) != 2 {
		t.Fatalf("expected two deltas, got %#v", deltas)
	}
	if deltas[0] != (trafficDelta{Resource: "user", Tag: "alice", Up: 17, Down: 3}) {
		t.Fatalf("unexpected user delta: %#v", deltas[0])
	}
	if deltas[1] != (trafficDelta{Resource: "inbound", Tag: "mixed-in", Down: 4}) {
		t.Fatalf("unexpected inbound delta: %#v", deltas[1])
	}
}

func TestPublishStatsRealtimePublishesOnlinesAndTrafficDelta(t *testing.T) {
	realtime.CloseAll("test_reset")
	t.Cleanup(func() { realtime.CloseAll("test_done") })
	ch := make(chan realtime.Event, 2)
	unregister := realtime.Register(&realtime.ClientHandle{
		User:   "admin",
		Scope:  realtime.ScopeAdmin,
		SendCh: ch,
	})
	defer unregister()

	publishStatsRealtime(onlines{User: []string{"alice"}}, []model.Stats{
		{Resource: "user", Tag: "alice", Direction: true, Traffic: 10},
	})

	onlinesEvent := expectRealtimeEvent(t, ch, realtime.TopicOnlines)
	if payload, ok := onlinesEvent.Payload.(onlines); !ok || len(payload.User) != 1 || payload.User[0] != "alice" {
		t.Fatalf("unexpected onlines payload: %#v", onlinesEvent.Payload)
	}
	deltaEvent := expectRealtimeEvent(t, ch, realtime.TopicTrafficDelta)
	deltas, ok := deltaEvent.Payload.([]trafficDelta)
	if !ok || len(deltas) != 1 || deltas[0].Up != 10 {
		t.Fatalf("unexpected delta payload: %#v", deltaEvent.Payload)
	}
}

func TestUpdateClientTrafficDeltasBatchesTwoHundredClients(t *testing.T) {
	initSettingTestDB(t)

	clients := make([]model.Client, 0, 200)
	deltas := make(map[string]clientTrafficDelta, 200)
	expectedUp := make(map[string]int64, 200)
	expectedDown := make(map[string]int64, 200)
	for i := 0; i < 200; i++ {
		name := fmt.Sprintf("client-%03d", i)
		initialUp := int64(i)
		initialDown := int64(i * 2)
		upDelta := int64(i + 10)
		downDelta := int64(i + 20)
		clients = append(clients, model.Client{
			Name: name,
			Up:   initialUp,
			Down: initialDown,
		})
		deltas[name] = clientTrafficDelta{
			up:   upDelta,
			down: downDelta,
		}
		expectedUp[name] = initialUp + upDelta
		expectedDown[name] = initialDown + downDelta
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.GetDB().Transaction(func(tx *gorm.DB) error {
		return updateClientTrafficDeltas(tx, deltas)
	}); err != nil {
		t.Fatal(err)
	}

	var stored []model.Client
	if err := database.GetDB().Order("name asc").Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored) != 200 {
		t.Fatalf("expected 200 clients, got %d", len(stored))
	}
	for _, client := range stored {
		if client.Up != expectedUp[client.Name] || client.Down != expectedDown[client.Name] {
			t.Fatalf("unexpected traffic for %s: up=%d down=%d", client.Name, client.Up, client.Down)
		}
	}
}

func expectRealtimeEvent(t *testing.T, ch <-chan realtime.Event, topic realtime.Topic) realtime.Event {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != topic {
			t.Fatalf("expected %s, got %s", topic, event.Type)
		}
		return event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", topic)
		return realtime.Event{}
	}
}
