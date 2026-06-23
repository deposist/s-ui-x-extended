package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x-extended/core"
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestConfigSaveTlsEditHotReloadsAffectedInboundsAndServicesWithoutCoreRestart(t *testing.T) {
	initSettingTestDB(t)

	coreInstance := core.NewCore()
	if err := coreInstance.Start([]byte(`{"log":{"disabled":true},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)); err != nil {
		t.Skipf("minimal core start unavailable for tls hot-reload regression: %v", err)
	}
	t.Cleanup(func() {
		_ = coreInstance.Stop()
	})

	tlsRow := model.Tls{
		Name:   "cert",
		Server: json.RawMessage(`{"enabled":true}`),
		Client: json.RawMessage(`{}`),
	}
	if err := database.GetDB().Create(&tlsRow).Error; err != nil {
		t.Fatal(err)
	}
	inbound := model.Inbound{
		Type:    "mixed",
		Tag:     "mixed-tls-reload",
		TlsId:   tlsRow.Id,
		Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`),
		OutJson: json.RawMessage(`{}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	svc := model.Service{
		Type:  "ssh",
		Tag:   "svc-tls-reload",
		TlsId: tlsRow.Id,
	}
	if err := database.GetDB().Create(&svc).Error; err != nil {
		t.Fatal(err)
	}

	var reloadedInbounds, reloadedServices []uint
	prevRestartInbounds := restartInboundsAfterSave
	restartInboundsAfterSave = func(_ *ConfigService, inboundIds []uint) error {
		reloadedInbounds = append([]uint(nil), inboundIds...)
		return nil
	}
	t.Cleanup(func() {
		restartInboundsAfterSave = prevRestartInbounds
	})
	prevRestartServices := restartServicesAfterSave
	restartServicesAfterSave = func(_ *ConfigService, serviceIds []uint) error {
		reloadedServices = append([]uint(nil), serviceIds...)
		return nil
	}
	t.Cleanup(func() {
		restartServicesAfterSave = prevRestartServices
	})

	before := coreInstance.GetInstance()
	tlsRow.Server = json.RawMessage(`{"enabled":true,"server_name":"renewed.example.com"}`)
	payload, err := json.Marshal(tlsRow)
	if err != nil {
		t.Fatal(err)
	}

	configService := NewConfigServiceWithRuntime(NewRuntime(coreInstance))
	objs, err := configService.Save("tls", "edit", payload, "", "admin", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(reloadedInbounds, []uint{inbound.Id}) {
		t.Fatalf("tls edit should hot-reload referencing inbound; got %v, want [%d]", reloadedInbounds, inbound.Id)
	}
	if !reflect.DeepEqual(reloadedServices, []uint{svc.Id}) {
		t.Fatalf("tls edit should hot-reload referencing service; got %v, want [%d]", reloadedServices, svc.Id)
	}
	if coreInstance.GetInstance() != before {
		t.Fatal("tls hot reload must not restart the whole core instance")
	}
	if !reflect.DeepEqual(objs, []string{"tls", "clients", "inbounds"}) {
		t.Fatalf("unexpected partial reload objects: %v", objs)
	}
}
