package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

type inboundOpsRecorder struct {
	ops []string
}

func (r *inboundOpsRecorder) stubInboundHooks(t *testing.T) {
	t.Helper()
	prevRestartInbounds := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, inboundIds []uint) error {
		r.ops = append(r.ops, fmt.Sprintf("reload-inbounds:%v", inboundIds))
		return nil
	}
	prevRemoveInbounds := removeInboundsFromCoreAfterSave
	removeInboundsFromCoreAfterSave = func(_ *ConfigService, tags []string) error {
		r.ops = append(r.ops, fmt.Sprintf("remove-inbounds:%v", tags))
		return nil
	}
	prevRestartServices := restartServicesAfterSave
	restartServicesAfterSave = func(_ *ConfigService, serviceIds []uint) error {
		r.ops = append(r.ops, fmt.Sprintf("reload-services:%v", serviceIds))
		return nil
	}
	t.Cleanup(func() {
		restartInboundsAfterSave = prevRestartInbounds
		removeInboundsFromCoreAfterSave = prevRemoveInbounds
		restartServicesAfterSave = prevRestartServices
	})
}

func createTestInbound(t *testing.T, tag string) model.Inbound {
	t.Helper()
	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     tag,
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	return inbound
}

func createSsmService(t *testing.T, tag string, inboundTag string) model.Service {
	t.Helper()
	svc := model.Service{
		Type:    "ssm-api",
		Tag:     tag,
		Options: json.RawMessage(fmt.Sprintf(`{"listen":"127.0.0.1","listen_port":0,"servers":{"/main":%q}}`, inboundTag)),
	}
	if err := database.GetDB().Create(&svc).Error; err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestConfigSaveInboundsEditHotReloadsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	inbound := createTestInbound(t, "inb-hot-edit")

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"mixed","tag":"inb-hot-edit","listen":"127.0.0.1","listen_port":1}`, inbound.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload-inbounds:[%d]", inbound.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("inbound edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("inbound edit must not restart the whole core instance")
	}
	if !reflect.DeepEqual(objs, []string{"inbounds", "clients"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

func TestConfigSaveInboundsNewHotAddsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(`{"type":"mixed","tag":"inb-hot-new","listen":"127.0.0.1","listen_port":0}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	var created model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).Where("tag = ?", "inb-hot-new").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{fmt.Sprintf("reload-inbounds:[%d]", created.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("inbound create core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("inbound create must not restart the whole core instance")
	}
}

func TestConfigSaveInboundsRenameRemovesOldTagThenReloads(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	inbound := createTestInbound(t, "inb-old-name")

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"mixed","tag":"inb-new-name","listen":"127.0.0.1","listen_port":0}`, inbound.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-inbounds:[inb-old-name]", fmt.Sprintf("reload-inbounds:[%d]", inbound.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("inbound rename core ops = %v, want %v", recorder.ops, want)
	}
}

// ssm-api services capture the managed inbound adapter at construction, so an
// inbound edit must reload the referencing service after the inbound itself.
func TestConfigSaveInboundsEditCascadesSsmServiceReload(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	inbound := createTestInbound(t, "ss-managed")
	svc := createSsmService(t, "ssm", "ss-managed")

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"mixed","tag":"ss-managed","listen":"127.0.0.1","listen_port":1}`, inbound.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		fmt.Sprintf("reload-inbounds:[%d]", inbound.Id),
		fmt.Sprintf("reload-services:[%d]", svc.Id),
	}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("inbound edit with ssm reference core ops = %v, want %v", recorder.ops, want)
	}
}

func TestConfigSaveInboundsDelBlockedBySsmReference(t *testing.T) {
	initSettingTestDB(t)
	createTestInbound(t, "ss-blocked")
	createSsmService(t, "ssm-guard", "ss-blocked")

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("inbounds", "del", json.RawMessage(`"ss-blocked"`), "", "admin", "example.com")
	if err == nil {
		t.Fatal("deleting an ssm-referenced inbound must be blocked")
	}
	for _, fragment := range []string{`inbound "ss-blocked"`, `ssm-api service "ssm-guard"`, `servers["/main"]`} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("guard error %q does not mention %q", err.Error(), fragment)
		}
	}

	var count int64
	if err := database.GetDB().Model(model.Inbound{}).Where("tag = ?", "ss-blocked").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("blocked delete must keep the inbound row")
	}
}

func TestConfigSaveInboundsRenameBlockedBySsmReference(t *testing.T) {
	initSettingTestDB(t)
	inbound := createTestInbound(t, "ss-rename-blocked")
	createSsmService(t, "ssm-rename-guard", "ss-rename-blocked")

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"mixed","tag":"ss-renamed","listen":"127.0.0.1","listen_port":0}`, inbound.Id))
	_, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com")
	if err == nil {
		t.Fatal("renaming an ssm-referenced inbound must be blocked")
	}
	if !strings.Contains(err.Error(), `ssm-api service "ssm-rename-guard"`) {
		t.Fatalf("guard error %q does not name the referencing service", err.Error())
	}

	var current model.Inbound
	if err := database.GetDB().Model(model.Inbound{}).Where("id = ?", inbound.Id).First(&current).Error; err != nil {
		t.Fatal(err)
	}
	if current.Tag != "ss-rename-blocked" {
		t.Fatalf("blocked rename must keep the old tag, got %q", current.Tag)
	}
}

func TestConfigSaveInboundsDelRemovesFromCoreWithoutRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	createTestInbound(t, "inb-hot-del")

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("inbounds", "del", json.RawMessage(`"inb-hot-del"`), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-inbounds:[inb-hot-del]"}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("inbound delete core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("inbound delete must not restart the whole core instance")
	}
	var count int64
	if err := database.GetDB().Model(model.Inbound{}).Where("tag = ?", "inb-hot-del").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("inbound row was not deleted")
	}
}

// Real end-to-end variant: the edit must reach the running core through the
// actual RestartInbounds path without swapping the core instance.
func TestConfigSaveInboundsEditAppliesToRunningCoreRealReload(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	inbound := createTestInbound(t, "inb-real-reload")

	before := coreInstance.GetInstance()
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"mixed","tag":"inb-real-reload","listen":"127.0.0.1","listen_port":0}`, inbound.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if coreInstance.GetInstance() != before {
		t.Fatal("real inbound hot reload must not restart the core (fallback full restart fired?)")
	}
	if err := coreInstance.RemoveInbound("inb-real-reload"); err != nil {
		t.Fatalf("inbound missing from running core after real hot reload: %v", err)
	}
}

func TestInboundSaveRejectsUnknownFieldBeforeDBCommit(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	payload := json.RawMessage(`{"type":"mixed","tag":"inb-unknown-field","listen":"127.0.0.1","listen_port":1,"bogus_top_level":true}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	before := coreInstance.GetInstance()
	if _, err := configService.Save("inbounds", "new", payload, "", "admin", "example.com"); err == nil {
		t.Fatal("expected inbound save to reject unknown field")
	} else if !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "bogus_top_level") {
		t.Fatalf("inbound save error = %v, want unknown field bogus_top_level", err)
	}

	db := database.GetDB()
	var count int64
	if err := db.Model(model.Inbound{}).Where("tag = ?", "inb-unknown-field").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rejected inbound persisted: %d rows", count)
	}
	if len(recorder.ops) != 0 {
		t.Fatalf("post-commit inbound hooks fired after rejection: %v", recorder.ops)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("rejected inbound save restarted the core")
	}
	if !coreInstance.IsRunning() {
		t.Fatal("core stopped after rejected inbound save")
	}
	var changes int64
	if err := db.Model(model.Changes{}).Count(&changes).Error; err != nil {
		t.Fatal(err)
	}
	if changes != 0 {
		t.Fatalf("audit Changes row committed after rejection: %d rows", changes)
	}
}

func TestInboundSaveRejectsUnknownUserFieldAfterClientAssignment(t *testing.T) {
	initSettingTestDB(t)
	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)

	db := database.GetDB()
	client := model.Client{
		Enable:   true,
		Name:     "trojan-client",
		Inbounds: json.RawMessage(`[]`),
		Config:   json.RawMessage(`{"trojan":{"password":"pw","bogus_user_field":true}}`),
		Links:    json.RawMessage(`[]`),
	}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	initialConfig := string(client.Config)

	payload := json.RawMessage(`{"type":"trojan","tag":"trojan-final-invalid","listen":"127.0.0.1","listen_port":1}`)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("inbounds", "new", payload, fmt.Sprintf("%d", client.Id), "admin", "example.com")
	if err == nil {
		t.Fatal("expected inbound save to reject an unknown final user field")
	} else if !strings.Contains(err.Error(), "unknown field") || !strings.Contains(err.Error(), "bogus_user_field") {
		t.Fatalf("inbound save error = %v, want unknown user field bogus_user_field", err)
	}

	var inboundCount int64
	if err := db.Model(model.Inbound{}).Where("tag = ?", "trojan-final-invalid").Count(&inboundCount).Error; err != nil {
		t.Fatal(err)
	}
	if inboundCount != 0 {
		t.Fatalf("rejected inbound persisted: %d rows", inboundCount)
	}

	var got model.Client
	if err := db.First(&got, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if string(got.Inbounds) != "[]" {
		t.Fatalf("client inbound assignment was not rolled back: %s", got.Inbounds)
	}
	if string(got.Config) != initialConfig {
		t.Fatalf("client config changed after rejected save: %s", got.Config)
	}
	if string(got.Links) != "[]" {
		t.Fatalf("client links changed after rejected save: %s", got.Links)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("post-commit inbound hooks fired after rejection: %v", recorder.ops)
	}
	var changes int64
	if err := db.Model(model.Changes{}).Count(&changes).Error; err != nil {
		t.Fatal(err)
	}
	if changes != 0 {
		t.Fatalf("audit Changes row committed after rejection: %d rows", changes)
	}
}
