package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func seedLegacyAWGSettings(t *testing.T, values map[string]string) {
	t.Helper()
	db := database.GetDB()
	for key, value := range values {
		setting := model.Setting{Key: key, Value: value}
		if err := db.Where("key = ?", key).Assign(setting).FirstOrCreate(&setting).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func createLegacyManagedEndpoint(t *testing.T, tag, address string) model.Endpoint {
	t.Helper()
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"address":     []string{address},
		"private_key": privateKey.String(),
		"listen_port": 51820,
		"peers":       []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := model.Endpoint{Type: "wireguard", Tag: tag, Options: raw}
	if err := database.GetDB().Create(&endpoint).Error; err != nil {
		t.Fatal(err)
	}
	return endpoint
}

func createLegacyDevice(t *testing.T, clientID uint, address string) model.AWGDevice {
	t.Helper()
	now := time.Now().Unix()
	device := model.AWGDevice{
		ClientId: clientID, Name: "phone", CryptoContext: make([]byte, awgCryptoContextSize),
		PublicKey: "pub", PrivateKeyEnc: []byte{1}, PSKEnc: []byte{1},
		IPv4Address: address, DesiredEnabled: true, SyncState: "in_sync", Provisioned: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := database.GetDB().Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	return device
}

func TestMigrateLegacyAWGSettingsConvertsEndpoint(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	seedLegacyAWGSettings(t, map[string]string{
		"awgEnabled": "true", "awgEndpointTag": "legacy-awg",
		"awgPublicEndpoint": "203.0.113.10:51820", "awgDNS": "1.1.1.1, 1.0.0.1",
		"awgDefaultDeviceLimit": "5",
	})
	endpoint := createLegacyManagedEndpoint(t, "legacy-awg", "10.77.0.1/16")
	client := model.Client{Enable: true, Name: "legacy"}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	device := createLegacyDevice(t, client.Id, "10.77.0.2")

	migrated, err := MigrateLegacyAWGSettings(db)
	if err != nil || !migrated {
		t.Fatalf("migration = (%v, %v)", migrated, err)
	}

	var converted model.Endpoint
	if err := db.First(&converted, endpoint.Id).Error; err != nil {
		t.Fatal(err)
	}
	settings, err := AWGSettingsForEndpoint(converted)
	if err != nil {
		t.Fatalf("converted endpoint is not manageable: %v", err)
	}
	if settings.PublicEndpoint != "203.0.113.10:51820" || settings.DefaultDeviceLimit != 5 || len(settings.DNS) != 2 {
		t.Fatalf("converted metadata mismatch: %+v", settings)
	}
	var moved model.AWGDevice
	if err := db.First(&moved, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if moved.EndpointId != endpoint.Id {
		t.Fatalf("device endpoint_id = %d, want %d", moved.EndpointId, endpoint.Id)
	}
	var access model.ClientEndpointAccess
	if err := db.Where("client_id = ? AND endpoint_id = ?", client.Id, endpoint.Id).First(&access).Error; err != nil {
		t.Fatalf("client access row missing: %v", err)
	}
	if readLegacyAWGSetting(db, "awgEnabled") != "false" {
		t.Fatal("awgEnabled was not switched off")
	}

	// Second run is a no-op.
	migrated, err = MigrateLegacyAWGSettings(db)
	if err != nil || migrated {
		t.Fatalf("repeat migration = (%v, %v)", migrated, err)
	}
}

func TestMigrateLegacyAWGSettingsSkipsDisabled(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	seedLegacyAWGSettings(t, map[string]string{"awgEnabled": "false"})
	migrated, err := MigrateLegacyAWGSettings(db)
	if err != nil || migrated {
		t.Fatalf("migration = (%v, %v)", migrated, err)
	}
}

func TestMigrateLegacyAWGSettingsFailsWhenEndpointMissing(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	seedLegacyAWGSettings(t, map[string]string{"awgEnabled": "true", "awgEndpointTag": "ghost"})
	_, err := MigrateLegacyAWGSettings(db)
	if err == nil || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("missing endpoint error = %v", err)
	}
	if readLegacyAWGSetting(db, "awgEnabled") != "true" {
		t.Fatal("failed migration must leave the legacy settings untouched")
	}
}

func TestMigrateLegacyAWGSettingsRejectsDeviceOutsideSubnet(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	seedLegacyAWGSettings(t, map[string]string{"awgEnabled": "true", "awgEndpointTag": "legacy-awg", "awgPublicEndpoint": "203.0.113.10:51820"})
	createLegacyManagedEndpoint(t, "legacy-awg", "10.77.0.1/24")
	client := model.Client{Enable: true, Name: "legacy"}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	createLegacyDevice(t, client.Id, "10.77.99.5")
	_, err := MigrateLegacyAWGSettings(db)
	if err == nil || !strings.Contains(err.Error(), "outside the endpoint subnet") {
		t.Fatalf("subnet mismatch error = %v", err)
	}
	if readLegacyAWGSetting(db, "awgEnabled") != "true" {
		t.Fatal("failed migration must leave the legacy settings untouched")
	}
}

func TestMigrateLegacyAWGSettingsRepointsDevicesOfManagedEndpoint(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	seedLegacyAWGSettings(t, map[string]string{"awgEnabled": "true", "awgEndpointTag": "legacy-awg"})
	endpoint := createLegacyManagedEndpoint(t, "legacy-awg", "10.77.0.1/16")
	managedExt := `{"managed":true,"publicEndpoint":"203.0.113.10:51820","dns":["1.1.1.1"],"defaultDeviceLimit":3}`
	if err := db.Model(&model.Endpoint{}).Where("id = ?", endpoint.Id).Update("ext", json.RawMessage(managedExt)).Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: true, Name: "legacy"}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	device := createLegacyDevice(t, client.Id, "10.77.0.2")

	migrated, err := MigrateLegacyAWGSettings(db)
	if err != nil {
		t.Fatal(err)
	}
	if migrated {
		t.Fatal("already-managed endpoint must not count as converted")
	}
	var moved model.AWGDevice
	if err := db.First(&moved, device.Id).Error; err != nil {
		t.Fatal(err)
	}
	if moved.EndpointId != endpoint.Id {
		t.Fatalf("device endpoint_id = %d, want %d", moved.EndpointId, endpoint.Id)
	}
	var kept model.Endpoint
	if err := db.First(&kept, endpoint.Id).Error; err != nil {
		t.Fatal(err)
	}
	if string(kept.Ext) != managedExt {
		t.Fatalf("existing Ext metadata was rewritten: %s", kept.Ext)
	}
}
