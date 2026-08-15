package migration

import (
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func insertEndpoint(t *testing.T, db *gorm.DB, id int, endpointType string, options string) {
	t.Helper()
	if err := db.Exec("INSERT INTO endpoints(id, type, options) VALUES(?, ?, ?)", id, endpointType, options).Error; err != nil {
		t.Fatal(err)
	}
}

func loadEndpointOptions(t *testing.T, db *gorm.DB, id int) map[string]json.RawMessage {
	t.Helper()
	var options string
	if err := db.Raw("SELECT options FROM endpoints WHERE id = ?", id).Scan(&options).Error; err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(options), &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestTo18MigratesAWG20AmneziaTo30(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`CREATE TABLE endpoints (id integer PRIMARY KEY AUTOINCREMENT, type text, options blob)`).Error; err != nil {
		t.Fatal(err)
	}

	legacy := `{
	  "address": ["10.77.0.1/29"],
	  "private_key": "abc",
	  "amnezia": {
	    "jc": 3, "jmin": 10, "jmax": 20, "s1": 15, "s2": 18, "s3": 12, "s4": 8,
	    "h1": "1000-1099", "h2": 2000, "h3": 3000, "h4": 4000,
	    "i1": "<b 0x01020304><r 8>",
	    "j1": "<b 0x11><r 4>", "j2": "<c><r 6>", "j3": "<t><r 2>", "itime": 120
	  }
	}`
	insertEndpoint(t, db, 1, "wireguard", legacy)
	// Plain WireGuard without amnezia: untouched.
	insertEndpoint(t, db, 2, "wireguard", `{"address":["10.0.0.1/24"],"private_key":"def","peers":[]}`)
	// Non-wireguard endpoint: untouched even with amnezia-like keys.
	insertEndpoint(t, db, 3, "shadowtls", `{"amnezia":{"j1":"x","itime":9}}`)
	// WARP endpoint with legacy shared-schema fields (s/h/key) that the 2.6.x
	// WARPAmnezia schema no longer carries: stripped, timing defaults added.
	warpLegacy := `{
	  "address": ["10.0.0.2/32"],
	  "private_key": "warp-key",
	  "amnezia": {
	    "jc": 4, "jmin": 40, "jmax": 90,
	    "s1": 15, "s2": 18, "s3": 12, "s4": 8,
	    "h1": "1000-1099", "h2": 2000, "h3": 3000, "h4": 4000,
	    "i1": "<b 0x01020304><r 8>",
	    "header_protection_key": "c2hvcnQ="
	  }
	}`
	insertEndpoint(t, db, 4, "warp", warpLegacy)

	if err := to1_8(db); err != nil {
		t.Fatal(err)
	}

	raw := loadEndpointOptions(t, db, 1)
	var amnezia map[string]json.RawMessage
	if err := json.Unmarshal(raw["amnezia"], &amnezia); err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []string{"j1", "j2", "j3", "itime"} {
		if _, present := amnezia[legacy]; present {
			t.Fatalf("legacy field %s survived migration", legacy)
		}
	}
	for key, want := range awg30MigrationDefaults {
		got := string(amnezia[key])
		wantJSON, _ := json.Marshal(want)
		if got != string(wantJSON) {
			t.Fatalf("3.0 field %s = %s, want %s", key, got, wantJSON)
		}
	}
	// Existing 2.0 fields must survive.
	for _, key := range []string{"jc", "jmin", "jmax", "s1", "s2", "s3", "s4", "h1", "h2", "h3", "h4", "i1"} {
		if _, present := amnezia[key]; !present {
			t.Fatalf("existing field %s was dropped", key)
		}
	}

	// WARP row: legacy s/h/key stripped, timing defaults present, junk kept.
	warpRaw := loadEndpointOptions(t, db, 4)
	var warpAmnezia map[string]json.RawMessage
	if err := json.Unmarshal(warpRaw["amnezia"], &warpAmnezia); err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []string{"s1", "s2", "s3", "s4", "h1", "h2", "h3", "h4", "header_protection_key"} {
		if _, present := warpAmnezia[legacy]; present {
			t.Fatalf("warp legacy field %s survived migration", legacy)
		}
	}
	for _, key := range []string{"jc", "jmin", "jmax", "i1"} {
		if _, present := warpAmnezia[key]; !present {
			t.Fatalf("warp existing field %s was dropped", key)
		}
	}
	for key, want := range awg30MigrationDefaults {
		wantJSON, _ := json.Marshal(want)
		if string(warpAmnezia[key]) != string(wantJSON) {
			t.Fatalf("warp 3.0 field %s = %s, want %s", key, string(warpAmnezia[key]), wantJSON)
		}
	}

	// Idempotent: a second run changes nothing.
	before := loadEndpointOptions(t, db, 1)
	if err := to1_8(db); err != nil {
		t.Fatal(err)
	}
	after := loadEndpointOptions(t, db, 1)
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatal("second migration run rewrote options")
	}

	// Plain and non-wireguard rows are byte-identical.
	plain := loadEndpointOptions(t, db, 2)
	if _, has := plain["amnezia"]; has {
		t.Fatal("plain wireguard endpoint gained amnezia")
	}
	shadow := loadEndpointOptions(t, db, 3)
	var shadowAmnezia map[string]json.RawMessage
	if err := json.Unmarshal(shadow["amnezia"], &shadowAmnezia); err != nil {
		t.Fatal(err)
	}
	if string(shadowAmnezia["j1"]) != `"x"` || string(shadowAmnezia["itime"]) != "9" {
		t.Fatalf("non-wireguard endpoint was modified: %s", string(shadow["amnezia"]))
	}
}

func TestTo18SkipsAlreadyMigratedAndMalformed(t *testing.T) {
	db := openMigrationTestDB(t)
	if err := db.Exec(`CREATE TABLE endpoints (id integer PRIMARY KEY AUTOINCREMENT, type text, options blob)`).Error; err != nil {
		t.Fatal(err)
	}
	// Already-3.0 profile: no default insertion needed.
	modern := `{"amnezia":{"jc":4,"rekey_after_time":"120-180","content_padding_addition":"0"}}`
	insertEndpoint(t, db, 1, "wireguard", modern)
	// Malformed options row: must not fail the migration.
	insertEndpoint(t, db, 2, "wireguard", `{not-json`)

	if err := to1_8(db); err != nil {
		t.Fatal(err)
	}
	raw := loadEndpointOptions(t, db, 1)
	var amnezia map[string]json.RawMessage
	if err := json.Unmarshal(raw["amnezia"], &amnezia); err != nil {
		t.Fatal(err)
	}
	if string(amnezia["rekey_after_time"]) != `"120-180"` {
		t.Fatalf("modern endpoint was rewritten: %s", string(raw["amnezia"]))
	}
	var options string
	if err := db.Raw("SELECT options FROM endpoints WHERE id = ?", 2).Scan(&options).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(options, "not-json") {
		t.Fatal("malformed row was rewritten")
	}
}
