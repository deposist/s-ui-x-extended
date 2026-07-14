package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestLoadAWGManagedEndpoint(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	settings, err := parseAWGSettings(validAWGSettingValues())
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"address":     []string{"10.77.0.1/16"},
		"private_key": privateKey.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Endpoint{Type: "wireguard", Tag: settings.EndpointTag, Options: raw}).Error; err != nil {
		t.Fatal(err)
	}

	managed, err := LoadAWGManagedEndpoint(db, settings)
	if err != nil {
		t.Fatal(err)
	}
	if managed.ServerAddress.String() != "10.77.0.1" {
		t.Fatalf("ServerAddress = %s", managed.ServerAddress)
	}
	if managed.ServerPublicKey != privateKey.PublicKey().String() {
		t.Fatalf("ServerPublicKey = %q, want canonical public key", managed.ServerPublicKey)
	}
}

func TestLoadAWGManagedEndpointRejectsInvalidEndpointWithoutLeakingOptions(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    *model.Endpoint
		mutate      func(*AWGSettings)
		secret      string
		wantMessage string
	}{
		{name: "missing", wantMessage: "not found"},
		{name: "wrong type", endpoint: &model.Endpoint{Type: "direct", Options: []byte(`{}`)}, wantMessage: "must use WireGuard"},
		{name: "malformed options", endpoint: &model.Endpoint{Type: "wireguard", Options: []byte(`{"private_key":"secret-fragment"`)}, secret: "secret-fragment", wantMessage: "options are invalid"},
		{name: "invalid private key", endpoint: &model.Endpoint{Type: "wireguard", Options: []byte(`{"private_key":"secret-private","address":["10.77.0.1/16"]}`)}, secret: "secret-private", wantMessage: "private key is invalid"},
		{name: "missing address", endpoint: endpointWithManagedLoaderOptions(t, nil), wantMessage: "requires an IPv4 address"},
		{name: "network address", endpoint: endpointWithManagedLoaderOptions(t, []string{"10.77.0.0/16"}), wantMessage: "does not match awgSubnet"},
		{name: "broadcast address", endpoint: endpointWithManagedLoaderOptions(t, []string{"10.77.255.255/16"}), wantMessage: "does not match awgSubnet"},
		{name: "outside subnet", endpoint: endpointWithManagedLoaderOptions(t, []string{"10.78.0.1/16"}), wantMessage: "does not match awgSubnet"},
		{name: "empty tag", mutate: func(settings *AWGSettings) { settings.EndpointTag = "" }, wantMessage: "tag is empty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initSettingTestDB(t)
			db := database.GetDB()
			settings, err := parseAWGSettings(validAWGSettingValues())
			if err != nil {
				t.Fatal(err)
			}
			if test.mutate != nil {
				test.mutate(&settings)
			}
			if test.endpoint != nil {
				test.endpoint.Tag = settings.EndpointTag
				if err := db.Create(test.endpoint).Error; err != nil {
					t.Fatal(err)
				}
			}
			_, err = LoadAWGManagedEndpoint(db, settings)
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("error = %v, want message containing %q", err, test.wantMessage)
			}
			if test.secret != "" && strings.Contains(err.Error(), test.secret) {
				t.Fatalf("error leaked endpoint options: %v", err)
			}
		})
	}
}

func endpointWithManagedLoaderOptions(t *testing.T, addresses []string) *model.Endpoint {
	t.Helper()
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"address": addresses, "private_key": privateKey.String()})
	if err != nil {
		t.Fatal(err)
	}
	return &model.Endpoint{Type: "wireguard", Options: raw}
}
