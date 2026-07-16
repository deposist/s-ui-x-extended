package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

const (
	validSudokuHalfA = "0100000000000000000000000000000000000000000000000000000000000000"
	validSudokuHalfB = "0200000000000000000000000000000000000000000000000000000000000000"
	validSudokuKeyA  = validSudokuHalfA + validSudokuHalfB
	validSudokuKeyB  = validSudokuHalfB + validSudokuHalfA
)

func TestNormalizeSudokuSplitKey(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
		valid             bool
	}{
		{"empty fallback", "  ", "", true},
		{"lowercase", validSudokuKeyA, validSudokuKeyA, true},
		{"uppercase and whitespace", " \t" + strings.ToUpper(validSudokuKeyA) + "\n", validSudokuKeyA, true},
		{"master scalar is not split key", validSudokuHalfA, "", false},
		{"wrong length", validSudokuKeyA[:126], "", false},
		{"non hex", strings.Repeat("z", 128), "", false},
		{"noncanonical first", strings.Repeat("ff", 32) + validSudokuHalfA, "", false},
		{"noncanonical second", validSudokuHalfA + strings.Repeat("ff", 32), "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeSudokuSplitKey(tc.input)
			if (err == nil) != tc.valid || got != tc.want {
				t.Fatalf("got value=%q err=%v, want value=%q valid=%v", got, err, tc.want, tc.valid)
			}
			if err != nil && (err.Error() != invalidSudokuSplitKeyMessage || strings.Contains(err.Error(), tc.input)) {
				t.Fatalf("unsafe/unstable error: %q", err)
			}
		})
	}
}

func TestClientSaveValidatesAndNormalizesSudokuAcrossWritePaths(t *testing.T) {
	for _, act := range []string{"new", "addbulk"} {
		t.Run(act, func(t *testing.T) {
			initSettingTestDB(t)
			client := model.Client{Name: "sudoku-" + act, Enable: true, Inbounds: json.RawMessage(`[]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{"sudoku":{"key":"` + strings.ToUpper(validSudokuKeyA) + `"}}`)}
			var payload []byte
			if act == "addbulk" {
				payload, _ = json.Marshal([]*model.Client{&client})
			} else {
				payload, _ = json.Marshal(&client)
			}
			if _, err := (&ClientService{}).Save(database.GetDB(), act, payload, "example.com"); err != nil {
				t.Fatal(err)
			}
			var saved model.Client
			if err := database.GetDB().Where("name = ?", client.Name).First(&saved).Error; err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(saved.Config), validSudokuKeyA) || strings.Contains(string(saved.Config), strings.ToUpper(validSudokuKeyA)) && validSudokuKeyA != strings.ToUpper(validSudokuKeyA) {
				t.Fatalf("key not normalized: %s", saved.Config)
			}
		})
	}
}

func TestClientSaveRejectsInvalidSudokuOnDirectBackendPaths(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{Name: "bad", Enable: true, Inbounds: json.RawMessage(`[]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{"sudoku":{"key":"` + strings.Repeat("a", 64) + `"}}`)}
	payload, _ := json.Marshal(&client)
	if _, err := (&ClientService{}).Save(database.GetDB(), "new", payload, "example.com"); err == nil || err.Error() != invalidSudokuSplitKeyMessage {
		t.Fatalf("new accepted invalid key: %v", err)
	}

	client.Config = json.RawMessage(`{"sudoku":{"key":""}}`)
	payload, _ = json.Marshal(&client)
	if _, err := (&ClientService{}).Save(database.GetDB(), "new", payload, "example.com"); err != nil {
		t.Fatal(err)
	}
	client.Config = json.RawMessage(`{"sudoku":{"key":"` + strings.Repeat("f", 128) + `"}}`)
	payload, _ = json.Marshal([]*model.Client{&client})
	if _, err := (&ClientService{}).Save(database.GetDB(), "editbulk", payload, "example.com"); err == nil || err.Error() != invalidSudokuSplitKeyMessage {
		t.Fatalf("editbulk accepted invalid key: %v", err)
	}
}

func TestSudokuConfigRoundTripPreservesKeyForBackupRestore(t *testing.T) {
	original := model.Client{Name: "backup", Config: json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyA + `"}}`)}
	dump, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored model.Client
	if err := json.Unmarshal(dump, &restored); err != nil {
		t.Fatal(err)
	}
	if string(restored.Config) != string(original.Config) {
		t.Fatalf("Client.Config changed across backup-style JSON round trip: %s", restored.Config)
	}
	if err := normalizeClientSudokuConfig(&restored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restored.Config), validSudokuKeyA) {
		t.Fatalf("restored key was lost: %s", restored.Config)
	}
}

func TestClientSaveEditBulkPersistsSudokuKeyAndRebuildsLinks(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "sudoku", Tag: "bulk-sudoku", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443,"key":"INBOUND-KEY"}`), Addrs: json.RawMessage(`[]`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{Name: "bulk-key", Enable: true, Inbounds: json.RawMessage(fmt.Sprintf(`[%d]`, inbound.Id)), Config: json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyA + `"}}`), Links: json.RawMessage(`[{"type":"external","uri":"https://external.example/sub"}]`)}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	client.Config = json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyB + `"}}`)
	payload, _ := json.Marshal([]*model.Client{&client})
	ids, err := (&ClientService{}).Save(database.GetDB(), "editbulk", payload, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("delivery-only bulk edit scheduled core reload: %v", ids)
	}
	var saved model.Client
	if err := database.GetDB().First(&saved, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved.Config), validSudokuKeyB) {
		t.Fatalf("bulk key was not persisted: %s", saved.Config)
	}
	links := linkURIs(t, saved.Links)
	if !links["https://external.example/sub"] {
		t.Fatalf("external link lost: %s", saved.Links)
	}
	found := false
	for uri := range links {
		if !strings.HasPrefix(uri, "sudoku://") {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(uri, "sudoku://"))
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		found = decoded["k"] == validSudokuKeyB
	}
	if !found {
		t.Fatalf("bulk local link was not rebuilt with key B: %s", saved.Links)
	}
}

func TestSudokuOnlyConfigChangeDoesNotReportCoreInboundChanges(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{Name: "no-reload", Enable: true, Inbounds: json.RawMessage(`[]`), Links: json.RawMessage(`[]`), Config: json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyA + `"}}`)}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	client.Config = json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyB + `"}}`)
	payload, _ := json.Marshal(&client)
	ids, err := (&ClientService{}).Save(database.GetDB(), "edit", payload, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("Sudoku delivery-only edit scheduled inbound reload: %v", ids)
	}
}
