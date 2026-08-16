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

type endpointOpsRecorder struct {
	ops []string
}

func (r *endpointOpsRecorder) stubEndpointHooks(t *testing.T) {
	t.Helper()
	prevRestart := restartEndpointsAfterSave
	restartEndpointsAfterSave = func(_ *ConfigService, endpointIds []uint) error {
		r.ops = append(r.ops, fmt.Sprintf("reload-endpoints:%v", endpointIds))
		return nil
	}
	prevRemove := removeEndpointsFromCoreAfterSave
	removeEndpointsFromCoreAfterSave = func(_ *ConfigService, tags []string) error {
		r.ops = append(r.ops, fmt.Sprintf("remove-endpoints:%v", tags))
		return nil
	}
	t.Cleanup(func() {
		restartEndpointsAfterSave = prevRestart
		removeEndpointsFromCoreAfterSave = prevRemove
	})
}

// A fixed, parseable WireGuard key (test vector from the WireGuard docs) so a
// full core restart can rebuild the endpoint from the database when needed.
const testWireguardKey = "yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk="

func createTestEndpoint(t *testing.T, tag string) model.Endpoint {
	t.Helper()
	endpoint := model.Endpoint{
		Type: "wireguard",
		Tag:  tag,
		Options: json.RawMessage(fmt.Sprintf(
			`{"system":false,"address":["10.0.0.2/32"],"private_key":%q,"peers":[],"mtu":1408}`, testWireguardKey)),
	}
	if err := database.GetDB().Create(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	return endpoint
}

func endpointPayload(id uint, tag string, mtu int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"id":%d,"type":"wireguard","tag":%q,"system":false,"address":["10.0.0.2/32"],"private_key":%q,"peers":[],"mtu":%d}`,
		id, tag, testWireguardKey, mtu))
}

func TestConfigSaveWireGuardEndpointAdvancedFieldsRoundTrip(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload := json.RawMessage(fmt.Sprintf(
		`{"type":"wireguard","tag":"wg-advanced","system":false,"address":["10.0.0.2/32"],"private_key":%q,"peers":[],"mtu":1408,"disable_pauses":true,"preallocated_buffers_per_pool":8,"domain_strategy":"prefer_ipv4"}`,
		testWireguardKey))
	if _, err := configService.Save("endpoints", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("save endpoint: %v", err)
	}
	var row model.Endpoint
	if err := database.GetDB().Where("tag = ?", "wg-advanced").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	var opts map[string]any
	if err := json.Unmarshal(row.Options, &opts); err != nil {
		t.Fatal(err)
	}
	if opts["disable_pauses"] != true {
		t.Fatalf("disable_pauses = %#v, want true", opts["disable_pauses"])
	}
	if opts["preallocated_buffers_per_pool"] != float64(8) {
		t.Fatalf("preallocated_buffers_per_pool = %#v, want 8", opts["preallocated_buffers_per_pool"])
	}
	if opts["domain_strategy"] != "prefer_ipv4" {
		t.Fatalf("domain_strategy = %#v, want prefer_ipv4", opts["domain_strategy"])
	}
}

func TestConfigSaveEndpointsEditHotReloadsWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	endpoint := createTestEndpoint(t, "wg-hot-edit")

	recorder := &endpointOpsRecorder{}
	recorder.stubEndpointHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("endpoints", "edit", endpointPayload(endpoint.Id, "wg-hot-edit", 1400), "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload-endpoints:[%d]", endpoint.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("endpoint edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("unreferenced endpoint edit must not restart the whole core instance")
	}
	if !reflect.DeepEqual(objs, []string{"endpoints"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}

func TestManagedAWGEndpointPeersCannotBeEdited(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	managedExt := `{"managed":true,"publicEndpoint":"vpn.example.com:51820","dns":["1.1.1.1"],"defaultDeviceLimit":3}`
	endpoint := createTestEndpoint(t, "managed-awg")
	if err := db.Model(&model.Endpoint{}).Where("id = ?", endpoint.Id).
		Update("options", json.RawMessage(fmt.Sprintf(`{"system":false,"address":["10.77.0.1/24"],"private_key":%q,"listen_port":51820,"peers":[],"mtu":1408}`, testWireguardKey))).
		Update("ext", json.RawMessage(managedExt)).Error; err != nil {
		t.Fatal(err)
	}
	changedPeers := json.RawMessage(fmt.Sprintf(
		`{"id":%d,"type":"wireguard","tag":"managed-awg","system":false,"address":["10.77.0.1/24"],"private_key":%q,"listen_port":51820,"ext":%s,"peers":[{"public_key":%q,"allowed_ips":["10.77.0.2/32"]}],"mtu":1408}`,
		endpoint.Id, testWireguardKey, managedExt, testWireguardKey))
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("endpoints", "edit", changedPeers, "", "admin", "example.com")
	if err == nil || !strings.Contains(err.Error(), "peers are controlled") {
		t.Fatalf("managed peer edit error = %v", err)
	}
	unchanged := json.RawMessage(fmt.Sprintf(
		`{"id":%d,"type":"wireguard","tag":"managed-awg","system":false,"address":["10.77.0.1/24"],"private_key":%q,"listen_port":51820,"ext":%s,"peers":[],"mtu":1400}`,
		endpoint.Id, testWireguardKey, managedExt))
	if _, err := configService.Save("endpoints", "edit", unchanged, "", "admin", "example.com"); err != nil {
		t.Fatalf("non-peer managed endpoint edit was blocked: %v", err)
	}
	all, err := (&EndpointService{}).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(*all) != 1 || (*all)[0]["awgManaged"] != true {
		t.Fatalf("managed endpoint marker missing: %#v", all)
	}
}

// An endpoint referenced only by a route rule (lazy lookup) stays hot.
func TestConfigSaveEndpointsEditWithRouteRuleReferenceStaysHot(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	endpoint := createTestEndpoint(t, "wg-rule-ref")
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"route":{"rules":[{"network":"udp","outbound":"wg-rule-ref"}]}}`))

	recorder := &endpointOpsRecorder{}
	recorder.stubEndpointHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("endpoints", "edit", endpointPayload(endpoint.Id, "wg-rule-ref", 1400), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{fmt.Sprintf("reload-endpoints:[%d]", endpoint.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("route-rule-referenced endpoint edit core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("route-rule references are lazy; the edit must stay hot")
	}
}

// An endpoint referenced by an outbound's dial detour is captured eagerly at
// construction, so the edit must escalate to a full core restart.
func TestConfigSaveEndpointsEditWithDetourReferenceRestartsCore(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	endpoint := createTestEndpoint(t, "wg-detour-ref")
	outbound := model.Outbound{
		Type:    "socks",
		Tag:     "socks-via-wg",
		Options: json.RawMessage(`{"server":"127.0.0.1","server_port":1080,"detour":"wg-detour-ref"}`),
	}
	if err := database.GetDB().Create(&outbound).Error; err != nil {
		t.Fatal(err)
	}
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true}}`))

	recorder := &endpointOpsRecorder{}
	recorder.stubEndpointHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("endpoints", "edit", endpointPayload(endpoint.Id, "wg-detour-ref", 1400), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.ops) != 0 {
		t.Fatalf("eagerly referenced endpoint edit must not hot-reload, got ops %v", recorder.ops)
	}
	// Only the restart decision is asserted here: a local test build without
	// with_gvisor cannot actually run a netstack WireGuard endpoint, so the
	// rebuilt core may legitimately fail to come back up in this environment.
	if coreInstance.GetInstance() == before {
		t.Fatal("eagerly referenced endpoint edit must restart the core")
	}
}

func TestConfigSaveEndpointsRenameRemovesOldTagThenReloads(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	endpoint := createTestEndpoint(t, "wg-old-name")

	recorder := &endpointOpsRecorder{}
	recorder.stubEndpointHooks(t)

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("endpoints", "edit", endpointPayload(endpoint.Id, "wg-new-name", 1408), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-endpoints:[wg-old-name]", fmt.Sprintf("reload-endpoints:[%d]", endpoint.Id)}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("endpoint rename core ops = %v, want %v", recorder.ops, want)
	}
}

func TestConfigSaveEndpointsRenameBlockedByReference(t *testing.T) {
	initSettingTestDB(t)
	endpoint := createTestEndpoint(t, "wg-rename-blocked")
	outbound := model.Outbound{
		Type:    "socks",
		Tag:     "socks-pin",
		Options: json.RawMessage(`{"server":"127.0.0.1","server_port":1080,"detour":"wg-rename-blocked"}`),
	}
	if err := database.GetDB().Create(&outbound).Error; err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("endpoints", "edit", endpointPayload(endpoint.Id, "wg-renamed", 1408), "", "admin", "example.com")
	if err == nil {
		t.Fatal("renaming a referenced endpoint must be blocked")
	}
	for _, fragment := range []string{`endpoint "wg-rename-blocked"`, `outbound "socks-pin" (detour)`, "(e.g. direct)"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("guard error %q does not mention %q", err.Error(), fragment)
		}
	}

	var current model.Endpoint
	if err := database.GetDB().Model(model.Endpoint{}).Where("id = ?", endpoint.Id).First(&current).Error; err != nil {
		t.Fatal(err)
	}
	if current.Tag != "wg-rename-blocked" {
		t.Fatalf("blocked rename must keep the old tag, got %q", current.Tag)
	}
}

func TestConfigSaveEndpointsDelBlockedByReference(t *testing.T) {
	initSettingTestDB(t)
	createTestEndpoint(t, "wg-del-blocked")
	seedConfigBlob(t, json.RawMessage(`{"log":{"disabled":true},"route":{"final":"wg-del-blocked"}}`))

	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	_, err := configService.Save("endpoints", "del", json.RawMessage(`"wg-del-blocked"`), "", "admin", "example.com")
	if err == nil {
		t.Fatal("deleting an endpoint referenced by route.final must be blocked")
	}
	if !strings.Contains(err.Error(), "route final") {
		t.Fatalf("guard error %q does not mention route final", err.Error())
	}

	var count int64
	if err := database.GetDB().Model(model.Endpoint{}).Where("tag = ?", "wg-del-blocked").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("blocked delete must keep the endpoint row")
	}
}

func TestConfigSaveEndpointsDelRemovesFromCoreWithoutRestart(t *testing.T) {
	initSettingTestDB(t)
	coreInstance := startTestCore(t)
	createTestEndpoint(t, "wg-hot-del")

	recorder := &endpointOpsRecorder{}
	recorder.stubEndpointHooks(t)

	before := coreInstance.GetInstance()
	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	if _, err := configService.Save("endpoints", "del", json.RawMessage(`"wg-hot-del"`), "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}

	want := []string{"remove-endpoints:[wg-hot-del]"}
	if !reflect.DeepEqual(recorder.ops, want) {
		t.Fatalf("endpoint delete core ops = %v, want %v", recorder.ops, want)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("endpoint delete must not restart the whole core instance")
	}
	var count int64
	if err := database.GetDB().Model(model.Endpoint{}).Where("tag = ?", "wg-hot-del").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("endpoint row was not deleted")
	}
}
