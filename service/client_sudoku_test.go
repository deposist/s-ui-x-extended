package service

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"filippo.io/edwards25519"
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

func TestClientSaveGeneratesDistinctSudokuKeysPerInbound(t *testing.T) {
	initSettingTestDB(t)
	inbounds := []model.Inbound{
		{Type: "sudoku", Tag: "sudoku-a", Options: json.RawMessage(`{"key":"` + validSudokuHalfA + `"}`), Addrs: json.RawMessage(`[]`)},
		{Type: "sudoku", Tag: "sudoku-b", Options: json.RawMessage(`{"key":"` + validSudokuHalfB + `"}`), Addrs: json.RawMessage(`[]`)},
	}
	for i := range inbounds {
		if err := database.GetDB().Create(&inbounds[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	client := model.Client{
		Name: "auto-sudoku", Enable: true,
		Inbounds: json.RawMessage(fmt.Sprintf(`[%d,%d]`, inbounds[0].Id, inbounds[1].Id)),
		Config:   json.RawMessage(`{"sudoku":{"key":""}}`), Links: json.RawMessage(`[]`),
	}
	payload, _ := json.Marshal(&client)
	if _, err := (&ClientService{}).Save(database.GetDB(), "new", payload, "example.com"); err != nil {
		t.Fatal(err)
	}
	var saved model.Client
	if err := database.GetDB().Where("name = ?", client.Name).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	var config struct {
		Sudoku struct {
			Key  string            `json:"key"`
			Keys map[string]string `json:"keys"`
		} `json:"sudoku"`
	}
	if err := json.Unmarshal(saved.Config, &config); err != nil {
		t.Fatal(err)
	}
	if config.Sudoku.Key != "" || len(config.Sudoku.Keys) != 2 {
		t.Fatalf("unexpected generated config: %s", saved.Config)
	}
	for i, inbound := range inbounds {
		key := config.Sudoku.Keys[fmt.Sprint(inbound.Id)]
		if _, err := normalizeSudokuSplitKey(key); err != nil {
			t.Fatalf("inbound %d generated invalid key: %v", inbound.Id, err)
		}
		decoded, _ := hex.DecodeString(key)
		left, _ := edwards25519.NewScalar().SetCanonicalBytes(decoded[:32])
		right, _ := edwards25519.NewScalar().SetCanonicalBytes(decoded[32:])
		master := edwards25519.NewScalar().Add(left, right)
		want := validSudokuHalfA
		if i == 1 {
			want = validSudokuHalfB
		}
		if hex.EncodeToString(master.Bytes()) != want {
			t.Fatalf("inbound %d split key does not recover its master", inbound.Id)
		}
	}
}

func TestEnsureSudokuInboundMasterKeyGeneratesCanonicalScalar(t *testing.T) {
	inbound := model.Inbound{Type: "sudoku", Options: json.RawMessage(`{"key":""}`)}
	if err := ensureSudokuInboundMasterKey(&inbound, true); err != nil {
		t.Fatal(err)
	}
	var options struct {
		Key       string `json:"key"`
		MasterKey string `json:"master_key"`
	}
	if err := json.Unmarshal(inbound.Options, &options); err != nil {
		t.Fatal(err)
	}
	master, err := parseSudokuMasterKey(options.MasterKey)
	if err != nil {
		t.Fatalf("generated inbound master key is invalid: %v", err)
	}
	if options.Key != sudokuPublicKey(master) {
		t.Fatal("inbound public key does not match generated master key")
	}
}

func TestEnsureClientSudokuKeysRotatesChangedMasterAndPrunesRemovedInbound(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "sudoku", Tag: "rotate", Options: json.RawMessage(`{"master_key":"` + validSudokuHalfA + `","key":"public"}`), Addrs: json.RawMessage(`[]`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{Inbounds: json.RawMessage(fmt.Sprintf(`[%d]`, inbound.Id)), Config: json.RawMessage(`{"sudoku":{"keys":{"999":"` + validSudokuKeyA + `"}}}`)}
	if err := ensureClientSudokuKeys(database.GetDB(), &client); err != nil {
		t.Fatal(err)
	}
	var first struct {
		Sudoku struct {
			Keys map[string]string `json:"keys"`
		} `json:"sudoku"`
	}
	_ = json.Unmarshal(client.Config, &first)
	keyA := first.Sudoku.Keys[fmt.Sprint(inbound.Id)]
	if keyA == "" || first.Sudoku.Keys["999"] != "" {
		t.Fatalf("generation/pruning failed: %s", client.Config)
	}
	inbound.Options = json.RawMessage(`{"master_key":"` + validSudokuHalfB + `","key":"public"}`)
	if err := database.GetDB().Model(&inbound).Update("options", inbound.Options).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureClientSudokuKeys(database.GetDB(), &client); err != nil {
		t.Fatal(err)
	}
	var second struct {
		Sudoku struct {
			Keys map[string]string `json:"keys"`
		} `json:"sudoku"`
	}
	_ = json.Unmarshal(client.Config, &second)
	keyB := second.Sudoku.Keys[fmt.Sprint(inbound.Id)]
	if keyB == keyA {
		t.Fatal("master-key rotation retained incompatible client split key")
	}
	masterB, _ := parseSudokuMasterKey(validSudokuHalfB)
	if !splitSudokuKeyMatchesMaster(keyB, masterB) {
		t.Fatal("rotated client split key does not match new master")
	}
}

func TestInboundCoreJSONOmitsSudokuMasterKey(t *testing.T) {
	initSettingTestDB(t)
	row := model.Inbound{Type: "sudoku", Tag: "safe-core", Options: json.RawMessage(`{"listen_port":443,"key":"public","master_key":"` + validSudokuHalfA + `"}`)}
	raw, err := inboundCoreJSON(&InboundService{}, database.GetDB(), row)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "master_key") || strings.Contains(string(raw), validSudokuHalfA) {
		t.Fatalf("core config leaked Sudoku master key: %s", raw)
	}
}
