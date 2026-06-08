package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

// linkURIs collects the "uri" field of every link entry in a client's Links
// column, so a test can assert which previously-stored links survived a
// regeneration regardless of any freshly generated local links.
func linkURIs(t *testing.T, raw json.RawMessage) map[string]bool {
	t.Helper()
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatalf("unmarshal links %q: %v", raw, err)
	}
	uris := make(map[string]bool, len(links))
	for _, link := range links {
		uris[link["uri"]] = true
	}
	return uris
}

func assertKept(t *testing.T, uris map[string]bool, kept, dropped []string) {
	t.Helper()
	for _, uri := range kept {
		if !uris[uri] {
			t.Errorf("expected link %q to be kept, got %v", uri, uris)
		}
	}
	for _, uri := range dropped {
		if uris[uri] {
			t.Errorf("expected link %q to be dropped, got %v", uri, uris)
		}
	}
}

// TestRebuildClientLinksNeverEmitsNull guards the NULL Links bug class: an empty
// result must marshal to `[]`, never `null`, and invalid stored links make the
// caller skip the client (ok=false).
func TestRebuildClientLinksNeverEmitsNull(t *testing.T) {
	keepAll := func(map[string]string) bool { return true }

	links, ok, err := rebuildClientLinks(1, json.RawMessage(`{}`), json.RawMessage(`[]`), nil, "host", keepAll, "test")
	if err != nil || !ok {
		t.Fatalf("rebuild with empty inputs: ok=%v err=%v", ok, err)
	}
	if string(links) != "[]" {
		t.Fatalf("empty rebuild must marshal to [], got %q", links)
	}

	if _, ok, _ := rebuildClientLinks(1, json.RawMessage(`{}`), json.RawMessage(`{bad`), nil, "host", keepAll, "test"); ok {
		t.Fatal("invalid stored links must report ok=false so the caller skips the client")
	}
}

// TestUpdateClientsOnInboundAddKeepsOtherInboundLinks pins the keep rule of
// UpdateClientsOnInboundAdd: drop every link of the added inbound's tag (local
// or not), keep links belonging to other inbounds.
func TestUpdateClientsOnInboundAddKeepsOtherInboundLinks(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "trojan", Tag: "in-A", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443}`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Name:     "add-client",
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{}`),
		Links: json.RawMessage(`[
			{"remark":"in-A","type":"local","uri":"drop-local-A"},
			{"remark":"in-A","type":"external","uri":"drop-ext-A"},
			{"remark":"in-B","type":"local","uri":"keep-local-B"},
			{"remark":"in-B","type":"external","uri":"keep-ext-B"}
		]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&ClientService{}).UpdateClientsOnInboundAdd(database.GetDB(), fmt.Sprintf("%d", client.Id), inbound.Id, "host"); err != nil {
		t.Fatal(err)
	}

	var got model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	assertKept(t, linkURIs(t, got.Links),
		[]string{"keep-local-B", "keep-ext-B"},
		[]string{"drop-local-A", "drop-ext-A"})
}

// TestUpdateLinksByInboundChangeKeepsNonLocalAndOtherTags pins the keep rule of
// UpdateLinksByInboundChange: drop local links of the new tag and the old tag;
// keep non-local links and local links of unrelated tags.
func TestUpdateLinksByInboundChangeKeepsNonLocalAndOtherTags(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "trojan", Tag: "new-tag", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443}`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Name:     "change-client",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`),
		Links: json.RawMessage(`[
			{"remark":"new-tag","type":"local","uri":"drop-new-local"},
			{"remark":"old-tag","type":"local","uri":"drop-old-local"},
			{"remark":"other","type":"local","uri":"keep-other-local"},
			{"remark":"new-tag","type":"external","uri":"keep-new-external"}
		]`),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	inbounds := []model.Inbound{inbound}
	if err := (&ClientService{}).UpdateLinksByInboundChange(database.GetDB(), &inbounds, "host", "old-tag"); err != nil {
		t.Fatal(err)
	}

	var got model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	assertKept(t, linkURIs(t, got.Links),
		[]string{"keep-other-local", "keep-new-external"},
		[]string{"drop-new-local", "drop-old-local"})
}

// TestUpdateLinksWithFixedInboundsKeepsNonLocal pins the keep rule of
// updateLinksWithFixedInbounds: drop all local links, keep all non-local links.
func TestUpdateLinksWithFixedInboundsKeepsNonLocal(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "trojan", Tag: "fix-tag", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443}`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := &model.Client{
		Name:     "fixed-client",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
		Config:   json.RawMessage(`{}`),
		Links: json.RawMessage(`[
			{"remark":"fix-tag","type":"local","uri":"drop-local"},
			{"remark":"whatever","type":"external","uri":"keep-external"}
		]`),
	}
	if err := database.GetDB().Create(client).Error; err != nil {
		t.Fatal(err)
	}

	// updateLinksWithFixedInbounds mutates the client structs in place (the
	// caller persists them), so assert on the in-memory Links.
	if err := (&ClientService{}).updateLinksWithFixedInbounds(database.GetDB(), []*model.Client{client}, "host"); err != nil {
		t.Fatal(err)
	}
	assertKept(t, linkURIs(t, client.Links),
		[]string{"keep-external"},
		[]string{"drop-local"})
}
