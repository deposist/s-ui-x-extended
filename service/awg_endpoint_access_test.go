package service

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func createManagedAWGEndpoint(t *testing.T, tag string, managed bool) model.Endpoint {
	t.Helper()
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	options, err := json.Marshal(map[string]any{
		"address": []string{"10.77.0.1/24"}, "private_key": privateKey.String(), "listen_port": 51820,
		"amnezia": map[string]any{"jc": 3, "jmin": 10, "jmax": 20, "s1": 15, "s2": 18, "s3": 12, "s4": 8,
			"h1": "1000-1099", "h2": 2000, "h3": 3000, "h4": 4000},
	})
	if err != nil {
		t.Fatal(err)
	}
	ext, err := json.Marshal(model.AWGEndpointMetadata{
		Managed: managed, PublicEndpoint: "vpn.example.com:51820",
		DNS: []string{"1.1.1.1"}, DefaultDeviceLimit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := model.Endpoint{Type: "wireguard", Tag: tag, Options: options, Ext: ext}
	if err := database.GetDB().Create(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	return endpoint
}

func TestAWGSettingsForEndpointFromMetadata(t *testing.T) {
	initSettingTestDB(t)
	endpoint := createManagedAWGEndpoint(t, "awg-meta", true)

	settings, err := AWGSettingsForEndpoint(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Enabled || settings.EndpointTag != "awg-meta" {
		t.Fatalf("settings = %+v", settings)
	}
	if settings.PublicEndpoint != "vpn.example.com:51820" {
		t.Fatalf("PublicEndpoint = %q", settings.PublicEndpoint)
	}
	if settings.Subnet.String() != "10.77.0.0/24" {
		t.Fatalf("Subnet = %s", settings.Subnet)
	}
	if settings.DefaultDeviceLimit != 3 {
		t.Fatalf("DefaultDeviceLimit = %d", settings.DefaultDeviceLimit)
	}
}

func TestAWGSettingsForEndpointRejectsInvalidMetadata(t *testing.T) {
	initSettingTestDB(t)
	tests := []struct {
		name   string
		mutate func(*model.AWGEndpointMetadata)
	}{
		{name: "missing public endpoint", mutate: func(m *model.AWGEndpointMetadata) { m.PublicEndpoint = "" }},
		{name: "missing port", mutate: func(m *model.AWGEndpointMetadata) { m.PublicEndpoint = "vpn.example.com" }},
		{name: "zero port", mutate: func(m *model.AWGEndpointMetadata) { m.PublicEndpoint = "vpn.example.com:0" }},
		{name: "empty dns", mutate: func(m *model.AWGEndpointMetadata) { m.DNS = nil }},
		{name: "invalid dns", mutate: func(m *model.AWGEndpointMetadata) { m.DNS = []string{"not-an-ip"} }},
		{name: "zero limit", mutate: func(m *model.AWGEndpointMetadata) { m.DefaultDeviceLimit = 0 }},
		{name: "excessive limit", mutate: func(m *model.AWGEndpointMetadata) { m.DefaultDeviceLimit = 101 }},
	}
	endpoint := createManagedAWGEndpoint(t, "awg-invalid", true)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metadata := model.AWGEndpointMetadata{
				Managed: true, PublicEndpoint: "vpn.example.com:51820",
				DNS: []string{"1.1.1.1"}, DefaultDeviceLimit: 3,
			}
			test.mutate(&metadata)
			ext, err := json.Marshal(metadata)
			if err != nil {
				t.Fatal(err)
			}
			endpoint.Ext = ext
			if _, err := AWGSettingsForEndpoint(endpoint); err == nil {
				t.Fatal("expected metadata validation error")
			}
		})
	}
}

func TestEffectiveAWGEndpointAccessDeniesUnassignedClient(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	endpoint := createManagedAWGEndpoint(t, "awg-access", true)

	if _, _, _, err := EffectiveAWGEndpointAccess(db, 42, endpoint.Id); !errors.Is(err, ErrAWGEndpointAccessDenied) {
		t.Fatalf("unassigned access error = %v", err)
	}

	access := model.ClientEndpointAccess{ClientId: 42, EndpointId: endpoint.Id, Source: "manual", CreatedAt: 1, UpdatedAt: 1}
	if err := db.Create(&access).Error; err != nil {
		t.Fatal(err)
	}
	_, settings, limit, err := EffectiveAWGEndpointAccess(db, 42, endpoint.Id)
	if err != nil {
		t.Fatal(err)
	}
	if settings.EndpointTag != "awg-access" || limit != 3 {
		t.Fatalf("tag = %q limit = %d", settings.EndpointTag, limit)
	}

	if err := db.Model(&model.ClientEndpointAccess{}).Where("id = ?", access.Id).Update("device_limit", 7).Error; err != nil {
		t.Fatal(err)
	}
	_, _, limit, err = EffectiveAWGEndpointAccess(db, 42, endpoint.Id)
	if err != nil || limit != 7 {
		t.Fatalf("override limit = %d err = %v", limit, err)
	}
}

func TestReplaceClientAWGEndpointAccessRevokesRemovedAssignments(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	first := createManagedAWGEndpoint(t, "awg-a", true)
	second := createManagedAWGEndpoint(t, "awg-b", true)

	if err := ReplaceClientAWGEndpointAccess(db, 7, []uint{first.Id, second.Id}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.ClientEndpointAccess{}).Where("client_id = ?", 7).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("count = %d err = %v", count, err)
	}

	device := model.AWGDevice{
		ClientId: 7, EndpointId: second.Id, Name: "phone", CryptoContext: []byte{1},
		PublicKey: "revoke-on-unassign", PrivateKeyEnc: []byte{1}, PSKEnc: []byte{1},
		IPv4Address: "10.77.0.2", DesiredEnabled: true, SyncState: "in_sync", Provisioned: true,
		CreatedAt: 1, UpdatedAt: 1,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}

	if err := ReplaceClientAWGEndpointAccess(db, 7, []uint{first.Id}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ClientEndpointAccess{}).Where("client_id = ?", 7).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("count after removal = %d err = %v", count, err)
	}
	var stored model.AWGDevice
	if err := db.First(&stored, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DesiredEnabled || stored.SyncState != "pending_remove" {
		t.Fatalf("device not marked for removal: enabled=%v state=%q", stored.DesiredEnabled, stored.SyncState)
	}
}

func TestReplaceClientAWGEndpointAccessRejectsUnmanagedEndpoint(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	unmanaged := createManagedAWGEndpoint(t, "awg-plain", false)

	if err := ReplaceClientAWGEndpointAccess(db, 9, []uint{unmanaged.Id}); err == nil {
		t.Fatal("expected unmanaged endpoint assignment to fail")
	}
	if err := ReplaceClientAWGEndpointAccess(db, 9, []uint{unmanaged.Id + 100}); err == nil {
		t.Fatal("expected missing endpoint assignment to fail")
	}
}

func TestListManagedAWGEndpointsFiltersByMetadata(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	managed := createManagedAWGEndpoint(t, "awg-managed", true)
	createManagedAWGEndpoint(t, "awg-unmanaged", false)

	ids, err := ListManagedAWGEndpoints(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != managed.Id {
		t.Fatalf("ids = %v", ids)
	}
}
