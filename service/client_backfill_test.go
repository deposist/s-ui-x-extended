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
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
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

func TestTrustTunnelBackfillGeneratesPassword(t *testing.T) {
	initSettingTestDB(t)

	// Client with an EXISTING trusttunnel block but no password (legacy or
	// partially-created config). This reproduces issue #7: subscription JSON
	// had username but no password, so the official client connected but all
	// tunneled traffic failed with authorization failed.
	client := model.Client{
		Name:     "tt-no-pw",
		Enable:   true,
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{"trusttunnel":{"name":"tt-no-pw"}}`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	inbound := model.Inbound{
		Type: "trusttunnel",
		Tag:  "tt-new",
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}

	// First add: trusttunnel inbound assigned to client.
	if err := (&ClientService{}).UpdateClientsOnInboundAdd(database.GetDB(), fmt.Sprintf("%d", client.Id), inbound.Id, "host"); err != nil {
		t.Fatal(err)
	}

	var afterAdd model.Client
	if err := database.GetDB().First(&afterAdd, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	cfg := map[string]map[string]any{}
	if err := json.Unmarshal(afterAdd.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	pw1, ok := cfg["trusttunnel"]["password"].(string)
	if !ok || strings.TrimSpace(pw1) == "" {
		t.Fatalf("trusttunnel password not backfilled on add: %s", afterAdd.Config)
	}

	// Second edit cycle: existing block must keep its password (no regeneration).
	if err := (&ClientService{}).UpdateLinksByInboundChange(database.GetDB(), &[]model.Inbound{inbound}, "host", "tt-new"); err != nil {
		t.Fatal(err)
	}
	var afterEdit model.Client
	if err := database.GetDB().First(&afterEdit, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	cfg2 := map[string]map[string]any{}
	if err := json.Unmarshal(afterEdit.Config, &cfg2); err != nil {
		t.Fatal(err)
	}
	pw2 := cfg2["trusttunnel"]["password"].(string)
	if pw2 != pw1 {
		t.Fatalf("trusttunnel password changed on edit: before=%q after=%q", pw1, pw2)
	}
}
