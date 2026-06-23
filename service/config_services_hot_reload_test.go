package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

type recordedCoreOps struct {
	ops []string
}

func (r *recordedCoreOps) stubServiceHooks(t *testing.T) {
	t.Helper()
	prevRestart := restartServicesAfterSave
	restartServicesAfterSave = func(_ *ConfigService, serviceIds []uint) error {
		r.ops = append(r.ops, fmt.Sprintf("reload:%v", serviceIds))
		return nil
	}
	prevRemove := removeServicesFromCoreAfterSave
	removeServicesFromCoreAfterSave = func(_ *ConfigService, tags []string) error {
		r.ops = append(r.ops, fmt.Sprintf("remove:%v", tags))
		return nil
	}
	t.Cleanup(func() {
		restartServicesAfterSave = prevRestart
		removeServicesFromCoreAfterSave = prevRemove
	})
}

func createTestService(t *testing.T, tag string) model.Service {
	t.Helper()
	svc := model.Service{
		Type:    "derp",
		Tag:     tag,
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
	}
	if err := database.GetDB().Create(&svc).Error; err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestConfigSaveServicesEditHotReloadsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	svc := createTestService(t, "svc-hot-edit")

	recorder := &recordedCoreOps{}
	recorder.stubServiceHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"derp","tag":"svc-hot-edit","listen":"127.0.0.1","listen_port":1}`, svc.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("services", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload:[%d]", svc.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("service edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("service edit must not restart the whole core instance")
	}
	if !reflect.DeepEqual(objs, []string{"services"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

func TestConfigSaveServicesNewHotAddsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	recorder := &recordedCoreOps{}
	recorder.stubServiceHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(`{"type":"derp","tag":"svc-hot-new","listen":"127.0.0.1","listen_port":0}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("services", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	var created model.Service
	if err := database.GetDB().Model(model.Service{}).Where("tag = ?", "svc-hot-new").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{fmt.Sprintf("reload:[%d]", created.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("service create core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("service create must not restart the whole core instance")
	}
}

func TestConfigSaveServicesRenameRemovesOldTagThenReloads(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	svc := createTestService(t, "svc-old-name")

	recorder := &recordedCoreOps{}
	recorder.stubServiceHooks(t)

	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"derp","tag":"svc-new-name","listen":"127.0.0.1","listen_port":0}`, svc.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("services", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove:[svc-old-name]", fmt.Sprintf("reload:[%d]", svc.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("service rename core ops = %v, want %v", recorder.ops, want)
	}
}

func TestConfigSaveServicesDelRemovesFromCoreWithoutRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	createTestService(t, "svc-hot-del")

	recorder := &recordedCoreOps{}
	recorder.stubServiceHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("services", "del", json.RawMessage(`"svc-hot-del"`), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove:[svc-hot-del]"}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("service delete core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("service delete must not restart the whole core instance")
	}
	var count int64
	if err := database.GetDB().Model(model.Service{}).Where("tag = ?", "svc-hot-del").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("service row was not deleted")
	}
}

// A failed partial apply must fall back to a full core restart so the core
// never keeps serving a partially updated state. The delete flow is used here
// because after the fallback the core is rebuilt from the database, which no
// longer contains the service row.
func TestConfigSaveServicesApplyFailureFallsBackToRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	createTestService(t, "svc-apply-fail")
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	prevRemove := removeServicesFromCoreAfterSave
	removeServicesFromCoreAfterSave = func(_ *ConfigService, _ []string) error {
		return fmt.Errorf("simulated remove failure")
	}
	t.Cleanup(func() {
		removeServicesFromCoreAfterSave = prevRemove
	})

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("services", "del", json.RawMessage(`"svc-apply-fail"`), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("failed service apply must fall back to a full core restart")
	}
	if after == nil {
		t.Fatal("core did not come back up after the fallback restart")
	}
}
