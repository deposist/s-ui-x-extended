package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

func localLinkRemarks(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatalf("unmarshal links %q: %v", raw, err)
	}
	var remarks []string
	for _, l := range links {
		if l["type"] == "local" {
			remarks = append(remarks, l["remark"])
		}
	}
	return remarks
}

// TestUpdateLinksWithFixedInboundsUsesEachClientsOwnInbounds pins the M6 fix:
// updateLinksWithFixedInbounds must regenerate each client's local links from
// THAT client's own Inbounds, not from clients[0]. Before the fix, a multi-client
// save with heterogeneous inbound sets (act="editbulk") rebuilt every client's
// local links from the first client's inbounds, silently corrupting the
// subscriptions of clients with a different inbound set.
func TestUpdateLinksWithFixedInboundsUsesEachClientsOwnInbounds(t *testing.T) {
	initSettingTestDB(t)
	mkInbound := func(tag string) model.Inbound {
		in := model.Inbound{
			Type:    "trojan",
			Tag:     tag,
			Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443}`),
			Addrs:   json.RawMessage(`[]`),
		}
		if err := database.GetDB().Create(&in).Error; err != nil {
			t.Fatal(err)
		}
		return in
	}
	inA := mkInbound("in-A")
	inB := mkInbound("in-B")

	client1 := &model.Client{
		Name:     "het-A",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inA.Id)),
		Config:   json.RawMessage(`{"trojan":{"password":"pwA"}}`),
		Links:    json.RawMessage(`[]`),
	}
	client2 := &model.Client{
		Name:     "het-B",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inB.Id)),
		Config:   json.RawMessage(`{"trojan":{"password":"pwB"}}`),
		Links:    json.RawMessage(`[]`),
	}
	for _, cl := range []*model.Client{client1, client2} {
		if err := database.GetDB().Create(cl).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := (&ClientService{}).updateLinksWithFixedInbounds(database.GetDB(), []*model.Client{client1, client2}, "example.com"); err != nil {
		t.Fatal(err)
	}

	r1 := localLinkRemarks(t, client1.Links)
	r2 := localLinkRemarks(t, client2.Links)
	if len(r1) == 0 || len(r2) == 0 {
		t.Fatalf("each client must get its own local links; got r1=%v r2=%v (links1=%s links2=%s)", r1, r2, client1.Links, client2.Links)
	}
	for _, r := range r1 {
		if r != "in-A" {
			t.Fatalf("client1 got a local link for %q, want only in-A (the clients[0] bug); links=%s", r, client1.Links)
		}
	}
	for _, r := range r2 {
		if r != "in-B" {
			t.Fatalf("client2 got a local link for %q, want only in-B (the clients[0] bug); links=%s", r, client2.Links)
		}
	}
}
