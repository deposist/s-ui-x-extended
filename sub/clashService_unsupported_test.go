package sub

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"
)

// TestClashSubscriptionExplainsUnsupportedProtocol is the end-to-end guard for
// the delivery contract of the clash/clash-meta format: a node the format cannot
// carry must be left out AND the omission must be stated in the rendered config,
// while the sing-box JSON subscription - which does carry it - keeps the node.
// Without the reason an operator sees a node silently missing from Mihomo.
func TestClashSubscriptionExplainsUnsupportedProtocol(t *testing.T) {
	initSubTestDB(t)
	ClearSubscriptionOutputCache()
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	inbound := &model.Inbound{
		Type:    "ssh",
		Tag:     "ssh-node",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":2222}`),
		OutJson: json.RawMessage("null"),
		Addrs:   json.RawMessage(`[{"server":"example.com","server_port":2222}]`),
	}
	if err := util.FillOutJson(inbound, "example.com"); err != nil {
		t.Fatalf("FillOutJson(ssh): %v", err)
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}

	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "ssh-secret",
		Config:    json.RawMessage(`{"ssh":{"user":"alice","password":"s3cret"}}`),
		Inbounds:  json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Links:     json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	clashBody, _, err := (&ClashService{}).GetClash("ssh-secret")
	if err != nil {
		t.Fatalf("GetClash: %v", err)
	}
	if names := clashProxyNames(t, *clashBody); names["1.ssh-node"] || names["ssh-node"] {
		t.Errorf("clash subscription rendered a proxy it cannot express: %v", names)
	}
	if !strings.Contains(*clashBody, "ssh-node (ssh)") {
		t.Errorf("clash subscription does not explain the omitted node:\n%s", *clashBody)
	}

	jsonBody, _, err := (&JsonService{}).GetJson("ssh-secret", "")
	if err != nil {
		t.Fatalf("GetJson: %v", err)
	}
	var cfg struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(*jsonBody), &cfg); err != nil {
		t.Fatalf("JSON subscription body is not valid JSON: %v", err)
	}
	if findOutbound(cfg.Outbounds, "ssh") == nil {
		t.Fatalf("the JSON subscription must still carry the ssh node the reason points at: %s", *jsonBody)
	}
}