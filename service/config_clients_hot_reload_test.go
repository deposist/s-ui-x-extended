package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestConfigSaveClientsHotReloadsChangedInboundsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)

	coreInstance := core.NewCore()
	if err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)); err != nil {
		t.Skipf("minimal core start unavailable for client hot-reload regression: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})

	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     "mixed-hot-reload",
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inbounds, err := json.Marshal([]uint{inbound.Id})
	if err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:      true,
		Name:        "alice",
		Config:      json.RawMessage(`{"mixed":{"username":"alice","password":"old"}}`),
		Inbounds:    inbounds,
		Links:       json.RawMessage(`[]`),
		IPLimitMode: "monitor",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	var reloaded []uint
	prevRestart := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, inboundIds []uint) error {
		reloaded = append([]uint(nil), inboundIds...)
		return nil
	}
	t.Cleanup(func() {
		restartInboundsAfterSave = prevRestart
	})

	before := coreInstance.GetInstance()
	client.Config = json.RawMessage(`{"mixed":{"username":"alice","password":"new"}}`)
	payload, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("clients", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(reloaded, []uint{inbound.Id}) {
		t.Fatalf("changed client config should hot-reload affected inbound; got %v, want [%d]", reloaded, inbound.Id)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("client hot reload must not restart the whole core instance")
	}
	if !reflect.DeepEqual(objs, []string{"clients", "inbounds"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

func TestConfigSaveSudokuKeyRebuildsLinksWithoutInboundReload(t *testing.T) {
	initSettingTestDB(t)
	inbound := model.Inbound{Type: "sudoku", Tag: "sudoku-no-reload", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443,"key":"INBOUND-KEY"}`), Addrs: json.RawMessage(`[]`)}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: true, Name: "sudoku-client", Config: json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyA + `"}}`), Inbounds: json.RawMessage(fmt.Sprintf(`[%d]`, inbound.Id)), Links: json.RawMessage(`[{"type":"external","uri":"https://external.example/sub"}]`)}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	previousRestart := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, ids []uint) error {
		t.Fatalf("Sudoku key edit reloaded core inbounds: %v", ids)
		return nil
	}
	t.Cleanup(func() { restartInboundsAfterSave = previousRestart })

	client.Config = json.RawMessage(`{"sudoku":{"key":"` + validSudokuKeyB + `"}}`)
	payload, _ := json.Marshal(client)
	objs, err := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil)).Save("clients", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(objs, []string{"clients"}) {
		t.Fatalf("objects=%v, want clients only", objs)
	}

	var saved model.Client
	if err := database.GetDB().First(&saved, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	links := linkURIs(t, saved.Links)
	if !links["https://external.example/sub"] {
		t.Fatalf("external link lost: %s", saved.Links)
	}
	foundPersonal := false
	for uri := range links {
		if !strings.HasPrefix(uri, "sudoku://") {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(uri, "sudoku://"))
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		foundPersonal = payload["k"] == validSudokuKeyB
	}
	if !foundPersonal {
		t.Fatalf("personal local Sudoku link not rebuilt with B: %s", saved.Links)
	}
}

func TestConfigSaveClientsInvalidatesIPPolicyCacheWithoutInboundReload(t *testing.T) {
	initSettingTestDB(t)

	client := model.Client{
		Enable:      true,
		Name:        "alice",
		Config:      json.RawMessage(`{"mixed":{"username":"alice","password":"pw"}}`),
		Inbounds:    json.RawMessage(`[]`),
		Links:       json.RawMessage(`[]`),
		LimitIP:     1,
		IPLimitMode: "monitor",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	invalidations := 0
	prevInvalidate := invalidateClientPolicyCacheAfterSave
	invalidateClientPolicyCacheAfterSave = func() {
		invalidations++
	}
	t.Cleanup(func() {
		invalidateClientPolicyCacheAfterSave = prevInvalidate
	})

	prevRestart := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, inboundIds []uint) error {
		t.Fatalf("IP policy-only edit should not reload inbounds, got %v", inboundIds)
		return nil
	}
	t.Cleanup(func() {
		restartInboundsAfterSave = prevRestart
	})

	client.LimitIP = 2
	client.IPLimitMode = "enforce"
	payload, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	objs, err := configService.Save("clients", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if invalidations != 1 {
		t.Fatalf("client policy cache invalidations=%d, want 1", invalidations)
	}
	if !reflect.DeepEqual(objs, []string{"clients"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

// Unlike the stub-based regressions above, this test exercises the REAL
// post-commit reload path end to end: a client edit must reach the running
// core via RemoveInbound/AddInbound - without swapping the core instance and
// without the full-restart fallback firing.
func TestConfigSaveClientsEditAppliesToRunningCoreRealReload(t *testing.T) {
	initSettingTestDB(t)

	coreInstance := core.NewCore()
	if err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)); err != nil {
		t.Skipf("minimal core start unavailable for real reload regression: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})

	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     "mixed-real-reload",
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inbounds, err := json.Marshal([]uint{inbound.Id})
	if err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:      true,
		Name:        "bob",
		Config:      json.RawMessage(`{"mixed":{"username":"bob","password":"pw1"}}`),
		Inbounds:    inbounds,
		Links:       json.RawMessage(`[]`),
		IPLimitMode: "monitor",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	before := coreInstance.GetInstance()
	client.Config = json.RawMessage(`{"mixed":{"username":"bob","password":"pw2"}}`)
	payload, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("clients", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if coreInstance.GetInstance() != before {
		t.Fatal("real client hot reload must not restart the core (fallback full restart fired?)")
	}
	// The reloaded inbound must actually be registered in the running core:
	// a successful RemoveInbound proves AddInbound applied it.
	if err := coreInstance.RemoveInbound("mixed-real-reload"); err != nil {
		t.Fatalf("inbound missing from running core after real hot reload: %v", err)
	}
}
