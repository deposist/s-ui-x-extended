package importxui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

// TestBuildClientLinks is the CI regression guard for the migration bug where
// imported clients had a NULL Links column, so their inbounds never appeared in
// the subscription. It exercises buildClientLinks directly against a synthetic
// trojan/grpc inbound with no TLS — the exact shape that reproduced the report.
func TestBuildClientLinks(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()

	inbound := model.Inbound{
		Type:    "trojan",
		Tag:     "inbound-12223",
		TlsId:   0,
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(`{"server":"127.0.0.1","server_port":12223,"tag":"inbound-12223","transport":{"service_name":"hello","type":"grpc"},"type":"trojan"}`),
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":12223,"transport":{"service_name":"hello","type":"grpc"}}`),
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	config := json.RawMessage(`{"trojan":{"name":"AndPh","password":"jwbMqRgdLA"}}`)
	inboundsJSON, err := json.Marshal([]uint{inbound.Id})
	if err != nil {
		t.Fatal(err)
	}

	// With a hostname, a link is generated and baked with that host.
	raw, err := buildClientLinks(db, config, inboundsJSON, "panel.example.com")
	if err != nil {
		t.Fatalf("buildClientLinks: %v", err)
	}
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatalf("unmarshal links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1: %v", len(links), links)
	}
	got := links[0]
	if got["type"] != "local" || got["remark"] != "inbound-12223" {
		t.Errorf("unexpected link metadata: %v", got)
	}
	want := "trojan://jwbMqRgdLA@panel.example.com:12223?type=grpc&serviceName=hello#inbound-12223"
	if got["uri"] != want {
		t.Errorf("link uri = %q, want %q", got["uri"], want)
	}

	// Without a hostname, links are left nil (no broken empty-host link).
	if raw, err := buildClientLinks(db, config, inboundsJSON, ""); err != nil || raw != nil {
		t.Errorf("buildClientLinks(empty host) = (%s, %v), want (nil, nil)", raw, err)
	}
	// No inbounds means no links.
	if raw, err := buildClientLinks(db, config, json.RawMessage(`[]`), "panel.example.com"); err != nil || raw != nil {
		t.Errorf("buildClientLinks(no inbounds) = (%s, %v), want (nil, nil)", raw, err)
	}
}

// TestResolveLinkHostname guards the centralized fallback that keeps scheduled
// sync (and any host-less caller) from producing NULL links: an empty explicit
// hostname falls back to the destination's configured sub/web domain.
func TestResolveLinkHostname(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()

	// No domains configured, no explicit host -> empty.
	if got := resolveLinkHostname(db, ""); got != "" {
		t.Errorf("resolveLinkHostname(no settings) = %q, want \"\"", got)
	}
	// Explicit host always wins.
	if got := resolveLinkHostname(db, "explicit.example"); got != "explicit.example" {
		t.Errorf("resolveLinkHostname(explicit) = %q, want explicit.example", got)
	}
	// webDomain is used when set and no explicit host.
	if err := db.Create(&model.Setting{Key: "webDomain", Value: "web.example"}).Error; err != nil {
		t.Fatal(err)
	}
	if got := resolveLinkHostname(db, ""); got != "web.example" {
		t.Errorf("resolveLinkHostname(webDomain) = %q, want web.example", got)
	}
	// subDomain takes precedence over webDomain.
	if err := db.Create(&model.Setting{Key: "subDomain", Value: "sub.example"}).Error; err != nil {
		t.Fatal(err)
	}
	if got := resolveLinkHostname(db, ""); got != "sub.example" {
		t.Errorf("resolveLinkHostname(subDomain) = %q, want sub.example", got)
	}
}

// TestBuildMergedClientLinks guards that a merge regenerates local links over
// the merged inbound set while preserving the client's existing non-local
// (external/sub) links, and never clobbers links on a host-less import.
func TestBuildMergedClientLinks(t *testing.T) {
	initCompatDest(t)
	db := database.GetDB()

	inbound := model.Inbound{
		Type:    "trojan",
		Tag:     "inbound-12223",
		TlsId:   0,
		Addrs:   json.RawMessage(`[]`),
		OutJson: json.RawMessage(`{"server":"127.0.0.1","server_port":12223,"tag":"inbound-12223","transport":{"service_name":"hello","type":"grpc"},"type":"trojan"}`),
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":12223,"transport":{"service_name":"hello","type":"grpc"}}`),
	}
	if err := db.Create(&inbound).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	config := json.RawMessage(`{"trojan":{"name":"AndPh","password":"pw"}}`)
	inboundsJSON, _ := json.Marshal([]uint{inbound.Id})
	existing := json.RawMessage(`[{"remark":"manual","type":"external","uri":"vmess://external"},{"remark":"inbound-12223","type":"local","uri":"trojan://STALE@old:12223"}]`)

	raw, err := buildMergedClientLinks(db, config, inboundsJSON, "panel.example.com", existing)
	if err != nil {
		t.Fatalf("buildMergedClientLinks: %v", err)
	}
	var links []map[string]string
	if err := json.Unmarshal(raw, &links); err != nil {
		t.Fatal(err)
	}
	var local, external int
	for _, l := range links {
		switch l["type"] {
		case "local":
			local++
			if !strings.Contains(l["uri"], "panel.example.com") || strings.Contains(l["uri"], "STALE") {
				t.Errorf("stale/host-less local link survived: %s", l["uri"])
			}
		case "external":
			external++
			if l["uri"] != "vmess://external" {
				t.Errorf("external link mangled: %s", l["uri"])
			}
		}
	}
	if local != 1 || external != 1 {
		t.Errorf("got local=%d external=%d, want 1/1: %v", local, external, links)
	}

	// Host-less merge returns nil so the caller leaves existing Links intact.
	if raw, err := buildMergedClientLinks(db, config, inboundsJSON, "", existing); err != nil || raw != nil {
		t.Errorf("buildMergedClientLinks(empty host) = (%s, %v), want (nil, nil)", raw, err)
	}
}
