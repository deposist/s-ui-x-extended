package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// clientLinks decodes a client's stored links blob for assertions.
func clientLinks(t *testing.T, id uint) []map[string]string {
	t.Helper()
	var c model.Client
	if err := database.GetDB().Model(model.Client{}).Where("id = ?", id).First(&c).Error; err != nil {
		t.Fatalf("load client %d: %v", id, err)
	}
	var links []map[string]string
	if len(c.Links) > 0 {
		if err := json.Unmarshal(c.Links, &links); err != nil {
			t.Fatalf("decode links: %v", err)
		}
	}
	return links
}

func hasLocalLinkWithPrefix(links []map[string]string, prefix string) bool {
	for _, l := range links {
		if l["type"] == "local" && strings.HasPrefix(l["uri"], prefix) {
			return true
		}
	}
	return false
}

// TestRegenerateMissingLocalLinksAddsSudokuLink is the fix for issue #4's
// follow-up: a client assigned to a sudoku inbound before the link generator
// existed has no sudoku:// link, and only a manual re-save would add one. The
// startup backfill must add it without operator action.
func TestRegenerateMissingLocalLinksAddsSudokuLink(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "sudoku",
		Tag:     "sudoku-1",
		Addrs:   json.RawMessage(`[{"server":"example.com","server_port":8443}]`),
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:   true,
		Name:     "alice",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`),
		Links:    json.RawMessage(`[]`), // pre-upgrade: no sudoku link stored
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&ClientService{}).RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}

	links := clientLinks(t, client.Id)
	if !hasLocalLinkWithPrefix(links, "sudoku://") {
		t.Fatalf("backfill must add a sudoku:// local link, got %v", links)
	}
}

// TestRegenerateMissingLocalLinksIsIdempotent proves a second run makes no
// change: the client is untouched and the links stay identical.
func TestRegenerateMissingLocalLinksIsIdempotent(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "sudoku",
		Tag:     "sudoku-1",
		Addrs:   json.RawMessage(`[{"server":"example.com","server_port":8443}]`),
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:   true,
		Name:     "alice",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`),
		Links:    json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	svc := &ClientService{}
	if err := svc.RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}
	first := clientLinks(t, client.Id)

	if err := svc.RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}
	second := clientLinks(t, client.Id)

	if fmt.Sprintf("%v", first) != fmt.Sprintf("%v", second) {
		t.Fatalf("second run must be a no-op.\nfirst:  %v\nsecond: %v", first, second)
	}
}

// TestRegenerateMissingLocalLinksLeavesOtherClientsAlone proves a client with no
// sudoku/mieru inbound is never rewritten.
func TestRegenerateMissingLocalLinksLeavesOtherClientsAlone(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "shadowsocks",
		Tag:     "ss-1",
		Addrs:   json.RawMessage(`[{"server":"example.com","server_port":443}]`),
		Options: json.RawMessage(`{"listen_port":443}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	sentinel := json.RawMessage(`[{"type":"local","uri":"ss://sentinel","remark":"keep"}]`)
	client := model.Client{
		Enable:   true,
		Name:     "bob",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`),
		Links:    sentinel,
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&ClientService{}).RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}

	var got model.Client
	if err := database.GetDB().Model(model.Client{}).Where("id = ?", client.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if string(got.Links) != string(sentinel) {
		t.Fatalf("a client with no sudoku/mieru inbound must be left untouched.\n got: %s\nwant: %s", got.Links, sentinel)
	}
}

// TestRegenerateMissingLocalLinksNoBackfillInbounds proves the fast path: with no
// sudoku/mieru inbound at all, the function returns without scanning clients.
func TestRegenerateMissingLocalLinksNoBackfillInbounds(t *testing.T) {
	initSettingTestDB(t)
	if err := (&ClientService{}).RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatalf("no-op path must not error: %v", err)
	}
}

// TestRegenerateMissingLocalLinksMieruBackfillsCredsThenIsStable proves the mieru
// path: a client with no mieru credentials gets them generated (random) and a
// mierus:// link on the first run; the second run reuses the persisted creds and
// makes no further change, so a random password can't cause churn on every boot.
func TestRegenerateMissingLocalLinksMieruBackfillsCredsThenIsStable(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "mieru",
		Tag:     "mieru-1",
		Addrs:   json.RawMessage(`[{"server":"example.com"}]`),
		Options: json.RawMessage(`{"listen_ports":["2090:2099"],"transport":"TCP"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:   true,
		Name:     "carol",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`), // no mieru creds yet
		Links:    json.RawMessage(`[]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	svc := &ClientService{}
	if err := svc.RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}
	if !hasLocalLinkWithPrefix(clientLinks(t, client.Id), "mierus://") {
		t.Fatalf("mieru backfill must add a mierus:// local link, got %v", clientLinks(t, client.Id))
	}
	first := clientLinks(t, client.Id)

	if err := svc.RegenerateMissingLocalLinks("example.com"); err != nil {
		t.Fatal(err)
	}
	second := clientLinks(t, client.Id)
	if fmt.Sprintf("%v", first) != fmt.Sprintf("%v", second) {
		t.Fatalf("second run must reuse persisted creds and not churn.\nfirst:  %v\nsecond: %v", first, second)
	}
}
