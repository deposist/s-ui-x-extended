package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestInboundGetAllLoadsUsersForManyInboundsInBatch(t *testing.T) {
	initSettingTestDB(t)

	inbounds := make([]model.Inbound, 0, 50)
	for i := 0; i < 50; i++ {
		inbounds = append(inbounds, model.Inbound{
			Type:    "vmess",
			Tag:     fmt.Sprintf("vmess-%02d", i),
			Options: json.RawMessage(`{}`),
		})
	}
	if err := database.GetDB().Create(&inbounds).Error; err != nil {
		t.Fatal(err)
	}

	inboundIDs := make([]uint, 0, len(inbounds))
	expectedIDs := make(map[uint]bool, len(inbounds))
	for _, inbound := range inbounds {
		inboundIDs = append(inboundIDs, inbound.Id)
		expectedIDs[inbound.Id] = true
	}
	inboundIDsJSON, err := json.Marshal(inboundIDs)
	if err != nil {
		t.Fatal(err)
	}

	clients := make([]model.Client, 0, 100)
	for i := 0; i < 100; i++ {
		clients = append(clients, model.Client{
			Name:     fmt.Sprintf("client-%03d", i),
			Inbounds: json.RawMessage(inboundIDsJSON),
		})
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	got, err := (&InboundService{}).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	seen := 0
	for _, inbound := range *got {
		id, ok := inbound["id"].(uint)
		if !ok || !expectedIDs[id] {
			continue
		}
		users, ok := inbound["users"].([]string)
		if !ok {
			t.Fatalf("inbound %d users has unexpected type %T", id, inbound["users"])
		}
		if len(users) != 100 {
			t.Fatalf("inbound %d expected 100 users, got %d", id, len(users))
		}
		if users[0] != "client-000" || users[99] != "client-099" {
			t.Fatalf("inbound %d users are not in client order: first=%q last=%q", id, users[0], users[99])
		}
		seen++
	}
	if seen != 50 {
		t.Fatalf("expected 50 tested inbounds, got %d", seen)
	}
}

func TestAddUsersDropsTrojanTopLevelPassword(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "trojan",
		Tag:     "trojan-1",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443,"password":"hello"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Client{
		Name:     "alice",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
	}).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON, err := json.Marshal(map[string]any{
		"type": "trojan", "tag": "trojan-1",
		"listen": "0.0.0.0", "listen_port": 443, "password": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "trojan")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	// sing-box's Trojan inbound has no top-level "password" (only "users") and
	// rejects the whole config if one is present.
	if _, has := got["password"]; has {
		t.Errorf("trojan inbound must not keep a top-level password: %s", out)
	}
	if _, has := got["users"]; !has {
		t.Errorf("trojan inbound should have users injected: %s", out)
	}
}

func TestAddUsersInjectsExtendedProtocolUsers(t *testing.T) {
	for _, tc := range []struct {
		inboundType string
		userConfig  string
		wantName    string
		wantSecret  string
		secretKey   string
	}{
		{inboundType: "mieru", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "trusttunnel", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "ssh", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "mtproxy", userConfig: `{"name":"alice","secret":"0123456789abcdef0123456789abcdef"}`, wantName: "alice", wantSecret: "0123456789abcdef0123456789abcdef", secretKey: "secret"},
	} {
		t.Run(tc.inboundType, func(t *testing.T) {
			initSettingTestDB(t)

			inbound := model.Inbound{
				Type:    tc.inboundType,
				Tag:     tc.inboundType + "-1",
				Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":2999}`),
			}
			if err := database.GetDB().Create(&inbound).Error; err != nil {
				t.Fatal(err)
			}
			config := json.RawMessage(fmt.Sprintf(`{"%s":%s}`, tc.inboundType, tc.userConfig))
			if err := database.GetDB().Create(&model.Client{
				Enable:   true,
				Name:     "alice",
				Config:   config,
				Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
			}).Error; err != nil {
				t.Fatal(err)
			}

			inboundJSON, err := json.Marshal(map[string]any{
				"type": tc.inboundType, "tag": tc.inboundType + "-1", "listen": "0.0.0.0", "listen_port": 2999,
			})
			if err != nil {
				t.Fatal(err)
			}
			out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, tc.inboundType)
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Users []map[string]string `json:"users"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			users := got.Users
			if len(users) != 1 || users[0]["name"] != tc.wantName || users[0][tc.secretKey] != tc.wantSecret {
				t.Fatalf("%s users were not injected correctly: %s", tc.inboundType, out)
			}
		})
	}
}

func TestFetchUsersByConditionRejectsUnsupportedInboundTypeBeforeSQL(t *testing.T) {
	_, err := (&InboundService{}).fetchUsersByCondition(nil, "vmess'); DROP TABLE clients; --", "1=1", map[string]interface{}{})
	if err == nil {
		t.Fatal("unsupported inbound type should be rejected before SQL execution")
	}
}

func TestFetchUsersByConditionRejectsUnexpectedJSONFieldBeforeSQL(t *testing.T) {
	const inboundType = "test-malicious-field"
	old, existed := userJSONField[inboundType]
	userJSONField[inboundType] = "vmess') FROM clients; --"
	t.Cleanup(func() {
		if existed {
			userJSONField[inboundType] = old
		} else {
			delete(userJSONField, inboundType)
		}
	})

	_, err := (&InboundService{}).fetchUsersByCondition(nil, inboundType, "1=1", map[string]interface{}{})
	if err == nil {
		t.Fatal("unexpected JSON field should be rejected before SQL execution")
	}
}
