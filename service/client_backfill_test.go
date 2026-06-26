package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestUpdateClientsOnInboundAddBackfillsProtocolConfig(t *testing.T) {
	initSettingTestDB(t)

	client := model.Client{
		Name:     "backfill-me",
		Enable:   true,
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{"vmess":{"id":"old"}}`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	inbound := model.Inbound{
		Type: "mieru",
		Tag:  "mieru-new",
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&ClientService{}).UpdateClientsOnInboundAdd(database.GetDB(), fmt.Sprintf("%d", client.Id), inbound.Id, "host"); err != nil {
		t.Fatal(err)
	}

	var updatedClient model.Client
	if err := database.GetDB().First(&updatedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedClient.Config), `"mieru"`) {
		t.Fatalf("config was not backfilled: %s", updatedClient.Config)
	}
	if !strings.Contains(string(updatedClient.Config), `"password"`) {
		t.Fatalf("backfilled config missing password: %s", updatedClient.Config)
	}

	// Verify addUsers works and extracts the generated user
	inboundJSON, _ := json.Marshal(map[string]any{"type": "mieru", "tag": "mieru-new", "listen": "0.0.0.0", "listen_port": 2999})
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "mieru")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Users []map[string]string `json:"users"`
	}
	json.Unmarshal(out, &got)
	if len(got.Users) != 1 || got.Users[0]["name"] != "backfill-me" {
		t.Fatalf("expected backfill-me to be extracted, got %s", out)
	}
}

func TestClientSaveEditBulkBackfillsProtocolConfig(t *testing.T) {
	initSettingTestDB(t)

	client := model.Client{
		Name:     "bulk-me",
		Enable:   true,
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{"vmess":{"id":"old"}}`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	inbound := model.Inbound{
		Type: "mieru",
		Tag:  "mieru-bulk",
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}

	inboundsJSON := fmt.Sprintf("[%d]", inbound.Id)
	clientJSON := fmt.Sprintf(`[{"id":%d,"name":"bulk-me","enable":true,"inbounds":%s,"config":{"vmess":{"id":"old"}}}]`, client.Id, inboundsJSON)

	if _, err := (&ClientService{}).Save(database.GetDB(), "editbulk", json.RawMessage(clientJSON), "host"); err != nil {
		t.Fatal(err)
	}

	var updatedClient model.Client
	if err := database.GetDB().First(&updatedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedClient.Config), `"mieru"`) {
		t.Fatalf("config was not backfilled: %s", updatedClient.Config)
	}
}
