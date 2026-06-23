package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

type outboundOpsRecorder struct {
	ops []string
}

func (r *outboundOpsRecorder) stubOutboundHooks(t *testing.T) {
	t.Helper()
	prevRestart := restartOutboundsAfterSave
	restartOutboundsAfterSave = func(_ *ConfigService, outboundIds []uint) error {
		r.ops = append(r.ops, fmt.Sprintf("reload-outbounds:%v", outboundIds))
		return nil
	}
	prevRemove := removeOutboundsFromCoreAfterSave
	removeOutboundsFromCoreAfterSave = func(_ *ConfigService, tags []string) error {
		r.ops = append(r.ops, fmt.Sprintf("remove-outbounds:%v", tags))
		return nil
	}
	t.Cleanup(func() {
		restartOutboundsAfterSave = prevRestart
		removeOutboundsFromCoreAfterSave = prevRemove
	})
}

func createTestOutbound(t *testing.T, tag string, port int) model.Outbound {
	t.Helper()
	outbound := model.Outbound{
		Type:    "socks",
		Tag:     tag,
		Options: json.RawMessage(fmt.Sprintf(`{"server":"127.0.0.1","server_port":%d}`, port)),
	}
	if err := database.GetDB().Create(&outbound).Error; err != nil {
		t.Fatal(err)
	}
	return outbound
}

func socksPayload(id uint, tag string, port int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"socks","tag":%q,"server":"127.0.0.1","server_port":%d}`, id, tag, port))
}

// The headline case: an outbound referenced only by route rules and
// route.final (both resolved lazily per connection) is replaced hot.
func TestConfigSaveOutboundsEditWithRouteReferencesStaysHot(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-routed", 1080)
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"route":{"final":"proxy-routed","rules":[{"network":"tcp","outbound":"proxy-routed"}]}}`))

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("outbounds", "edit", socksPayload(outbound.Id, "proxy-routed", 1081), "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload-outbounds:[%d]", outbound.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("route-referenced outbound edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("route rule / route.final references are lazy; the edit must stay hot")
	}
	if !reflect.DeepEqual(objs, []string{"outbounds"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

// Real end-to-end variant: the running core must receive the replacement via
// the actual RestartOutbounds path without a core swap.
func TestConfigSaveOutboundsEditAppliesToRunningCoreRealReload(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := core.NewCore()
	err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"},{"type":"socks","tag":"proxy-real","server":"127.0.0.1","server_port":1080}]}`))
	if err != nil {
		t.Skipf("minimal core start unavailable for real outbound reload regression: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})

	outbound := createTestOutbound(t, "proxy-real", 1080)
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"route":{"final":"proxy-real"}}`))

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", socksPayload(outbound.Id, "proxy-real", 1081), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if coreInstance.GetInstance() != before {
		t.Fatal("real outbound hot reload must not restart the core (fallback full restart fired?)")
	}
	// The replaced outbound must actually be registered in the running core:
	// a successful RemoveOutbound proves AddOutbound applied it.
	if err := coreInstance.RemoveOutbound("proxy-real"); err != nil {
		t.Fatalf("outbound missing from running core after real hot reload: %v", err)
	}
}

// A selector holds member adapter pointers from Start(), so editing a member
// must escalate to a full core restart.
func TestConfigSaveOutboundsEditSelectorMemberRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	member := createTestOutbound(t, "proxy-member", 1080)
	selector := model.Outbound{
		Type:    "selector",
		Tag:     "auto-group",
		Options: json.RawMessage(`{"outbounds":["proxy-member","direct"]}`),
	}
	if err := database.GetDB().Create(&selector).Error; err != nil {
		t.Fatal(err)
	}
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", socksPayload(member.Id, "proxy-member", 1081), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("selector-member edit must not hot-reload, got ops %v", recorder.ops)
	}
	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("selector-member edit must restart the core")
	}
	if after == nil {
		t.Fatal("core did not come back up after the restart")
	}
}

// A dns server detour builds its dialer at construction - eager, restart.
func TestConfigSaveOutboundsEditDnsDetourReferenceRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-dns", 1080)
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"dns":{"servers":[{"tag":"dns-remote","type":"udp","server":"1.1.1.1","detour":"proxy-dns"}]}}`))

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", socksPayload(outbound.Id, "proxy-dns", 1081), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("dns-detour-referenced outbound edit must not hot-reload, got ops %v", recorder.ops)
	}
	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("dns-detour-referenced outbound edit must restart the core")
	}
	if after == nil {
		t.Fatal("core did not come back up after the restart")
	}
}

// Editing the selector itself stays hot as long as nothing references its tag.
func TestConfigSaveOutboundsEditSelectorItselfStaysHot(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	selector := model.Outbound{
		Type:    "selector",
		Tag:     "group-self",
		Options: json.RawMessage(`{"outbounds":["direct"]}`),
	}
	if err := database.GetDB().Create(&selector).Error; err != nil {
		t.Fatal(err)
	}

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"selector","tag":"group-self","outbounds":["direct"],"interrupt_exist_connections":true}`, selector.Id))
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload-outbounds:[%d]", selector.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("selector self-edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("editing an unreferenced selector must not restart the core")
	}
}

func TestConfigSaveOutboundsNewHotAddsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(`{"type":"socks","tag":"proxy-new","server":"127.0.0.1","server_port":1080}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	var created model.Outbound
	if err := database.GetDB().Model(model.Outbound{}).Where("tag = ?", "proxy-new").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{fmt.Sprintf("reload-outbounds:[%d]", created.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("outbound create core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("unreferenced outbound create must not restart the core")
	}
}

// Creating an outbound whose tag is already referenced eagerly (e.g. a relay
// detour pointing at it) must restart so the waiting references bind to it.
func TestConfigSaveOutboundsNewWithEagerReferenceRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	relay := model.Outbound{
		Type:    "socks",
		Tag:     "relay-waiting",
		Options: json.RawMessage(`{"server":"127.0.0.1","server_port":1090,"detour":"proxy-awaited"}`),
	}
	if err := database.GetDB().Create(&relay).Error; err != nil {
		t.Fatal(err)
	}
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	payload := json.RawMessage(`{"type":"socks","tag":"proxy-awaited","server":"127.0.0.1","server_port":1080}`)
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("eagerly awaited outbound create must not hot-reload, got ops %v", recorder.ops)
	}
	after := coreInstance.GetInstance()
	if after == before {
		t.Fatal("eagerly awaited outbound create must restart the core")
	}
	if after == nil {
		t.Fatal("core did not come back up after the restart")
	}
}

func TestConfigSaveOutboundsRenameRemovesOldTagThenReloads(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	outbound := createTestOutbound(t, "proxy-old-name", 1080)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "edit", socksPayload(outbound.Id, "proxy-new-name", 1080), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-outbounds:[proxy-old-name]", fmt.Sprintf("reload-outbounds:[%d]", outbound.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("outbound rename core ops = %v, want %v", recorder.ops, want)
	}
}

func TestConfigSaveOutboundsRenameBlockedByReference(t *testing.T) {
	initSettingTestDB(t)
	outbound := createTestOutbound(t, "proxy-rename-blocked", 1080)
	selector := model.Outbound{
		Type:    "selector",
		Tag:     "pin-group",
		Options: json.RawMessage(`{"outbounds":["proxy-rename-blocked"]}`),
	}
	if err := database.GetDB().Create(&selector).Error; err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("outbounds", "edit", socksPayload(outbound.Id, "proxy-renamed", 1080), "", "admin", "example.com")
	if err == nil {
		t.Fatal("renaming a referenced outbound must be blocked")
	}
	for _, fragment := range []string{`outbound "proxy-rename-blocked"`, `selector "pin-group" (outbounds list)`, "(e.g. direct)"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("guard error %q does not mention %q", err.Error(), fragment)
		}
	}

	var current model.Outbound
	if err := database.GetDB().Model(model.Outbound{}).Where("id = ?", outbound.Id).First(&current).Error; err != nil {
		t.Fatal(err)
	}
	if current.Tag != "proxy-rename-blocked" {
		t.Fatalf("blocked rename must keep the old tag, got %q", current.Tag)
	}
}

// Lazy references (route rules) do not force restarts on edit, but they MUST
// block deletes: a dangling tag would fail the next core start entirely.
func TestConfigSaveOutboundsDelBlockedByRouteRuleReference(t *testing.T) {
	initSettingTestDB(t)
	createTestOutbound(t, "proxy-del-blocked", 1080)
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"route":{"rules":[{"network":"tcp","outbound":"proxy-del-blocked"}]}}`))

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("outbounds", "del", json.RawMessage(`"proxy-del-blocked"`), "", "admin", "example.com")
	if err == nil {
		t.Fatal("deleting a route-rule-referenced outbound must be blocked")
	}
	for _, fragment := range []string{`outbound "proxy-del-blocked"`, "route rule #0 (outbound)", "(e.g. direct)"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("guard error %q does not mention %q", err.Error(), fragment)
		}
	}

	var count int64
	if err := database.GetDB().Model(model.Outbound{}).Where("tag = ?", "proxy-del-blocked").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("blocked delete must keep the outbound row")
	}
}

func TestConfigSaveOutboundsDelRemovesFromCoreWithoutRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	createTestOutbound(t, "proxy-hot-del", 1080)

	recorder := &outboundOpsRecorder{}
	recorder.stubOutboundHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("outbounds", "del", json.RawMessage(`"proxy-hot-del"`), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-outbounds:[proxy-hot-del]"}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("outbound delete core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("outbound delete must not restart the whole core instance")
	}
	var count int64
	if err := database.GetDB().Model(model.Outbound{}).Where("tag = ?", "proxy-hot-del").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("outbound row was not deleted")
	}
}
