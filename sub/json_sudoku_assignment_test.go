package sub

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"
)

// TestJsonSubscriptionDeliversSudokuForAssignedClient is the end-to-end guard
// for issue #4: a client with a sudoku inbound id in client.Inbounds must
// receive the sudoku outbound (carrying the shared key) in its JSON
// subscription. Sudoku is keyless (no per-user credential objects), so
// assignment is the ONLY way the outbound can reach a client.
func TestJsonSubscriptionDeliversSudokuForAssignedClient(t *testing.T) {
	initSubTestDB(t)
	ClearSubscriptionOutputCache()
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}

	inbound := &model.Inbound{
		Type: "sudoku",
		Tag:  "sudoku-in",
		Options: json.RawMessage(
			`{"listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY","aead_method":"chacha20-poly1305"}`),
		OutJson: json.RawMessage("null"),
		Addrs:   json.RawMessage("[]"),
	}
	if err := util.FillOutJson(inbound, "example.com"); err != nil {
		t.Fatalf("FillOutJson(sudoku): %v", err)
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}

	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "sudoku-secret",
		Config:    json.RawMessage(`{}`),
		Inbounds:  json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Links:     json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	body, _, err := (&JsonService{}).GetJson("sudoku-secret", "")
	if err != nil {
		t.Fatalf("GetJson: %v", err)
	}

	var cfg struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(*body), &cfg); err != nil {
		t.Fatalf("subscription body is not valid JSON: %v", err)
	}
	sudoku := findOutbound(cfg.Outbounds, "sudoku")
	if sudoku == nil {
		t.Fatalf("assigned sudoku inbound must yield a sudoku outbound in the JSON subscription: %s", *body)
	}
	if sudoku["key"] != "SHARED-KEY" {
		t.Errorf("sudoku outbound must carry the shared key, got %v", sudoku["key"])
	}
	if sudoku["server"] != "example.com" {
		t.Errorf("sudoku outbound server should be example.com, got %v", sudoku["server"])
	}
}

func TestJsonSubscriptionSelectsSudokuKeyForInboundID(t *testing.T) {
	initSubTestDB(t)
	inbound := &model.Inbound{
		Type: "sudoku", Tag: "sudoku-personal", Options: json.RawMessage(`{"listen_port":8443,"key":"SHARED-KEY"}`),
		OutJson: json.RawMessage(`{"type":"sudoku","tag":"sudoku-personal","server":"example.com","server_port":8443,"key":"SHARED-KEY"}`),
		Addrs:   json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(inbound).Error; err != nil {
		t.Fatal(err)
	}
	personal := "01000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000"
	config := json.RawMessage(fmt.Sprintf(`{"sudoku":{"keys":{"%d":"%s"}}}`, inbound.Id, personal))
	outbounds, _, err := (&JsonService{}).getOutbounds(config, []*model.Inbound{inbound})
	if err != nil {
		t.Fatal(err)
	}
	if len(*outbounds) != 1 || (*outbounds)[0]["key"] != personal {
		t.Fatalf("JSON subscription did not select inbound-specific key: %#v", *outbounds)
	}
}
