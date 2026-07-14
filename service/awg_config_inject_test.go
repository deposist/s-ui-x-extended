package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestInjectAWGManagedEndpointPeersAddsPSKOnlyInMemory(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	if err := db.AutoMigrate(&model.AWGDevice{}); err != nil {
		t.Fatal(err)
	}
	master := bytes.Repeat([]byte{0x91}, 32)
	t.Setenv(awgEncryptionKeyEnv, base64.StdEncoding.EncodeToString(master))
	cipher, err := NewAWGCipherFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: true, Name: "inject", Expiry: time.Now().Add(time.Hour).Unix()}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	context := bytes.Repeat([]byte{0x44}, awgCryptoContextSize)
	psk := bytes.Repeat([]byte{0x33}, 32)
	pskEnc, err := cipher.Encrypt(client.Id, context, psk)
	if err != nil {
		t.Fatal(err)
	}
	device := model.AWGDevice{ClientId: client.Id, Name: "phone", CryptoContext: context, PublicKey: wgtypes.Key(bytes.Repeat([]byte{0x22}, 32)).String(), PrivateKeyEnc: []byte{1}, PSKEnc: pskEnc, IPv4Address: "10.77.0.2", DesiredEnabled: true, SyncState: "in_sync", Provisioned: true, CreatedAt: 1, UpdatedAt: 1}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	endpoints := []json.RawMessage{json.RawMessage(`{"type":"wireguard","tag":"awg","peers":[]}`)}
	got, err := injectAWGManagedEndpointPeers(db, AWGSettings{Enabled: true, EndpointTag: "awg", Subnet: netip.MustParsePrefix("10.77.0.0/29")}, endpoints)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got[0], []byte(base64.StdEncoding.EncodeToString(psk))) {
		t.Fatalf("in-memory endpoint lacks PSK: %s", got[0])
	}
}
