package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func validAWGSettingValues() map[string]string {
	return map[string]string{
		"awgEnabled":              "true",
		"awgEndpointTag":          "managed-awg",
		"awgPublicEndpoint":       "vpn.example.com:51820",
		"awgSubnet":               "10.77.0.0/16",
		"awgDNS":                  "1.1.1.1, 1.0.0.1",
		"awgDefaultDeviceLimit":   "3",
		"awgReconcileIntervalSec": "30",
		"awgStatsIntervalSec":     "60",
		"awgMTU":                  "0",
	}
}

func TestParseAWGSettings(t *testing.T) {
	settings, err := parseAWGSettings(validAWGSettingValues())
	if err != nil {
		t.Fatal(err)
	}
	if settings.PublicEndpoint != "vpn.example.com:51820" || settings.Subnet.String() != "10.77.0.0/16" || len(settings.DNS) != 2 {
		t.Fatalf("unexpected AWG settings: %+v", settings)
	}
}

func TestParseAWGSettingsRejectsInvalidBounds(t *testing.T) {
	for key, value := range map[string]string{
		"awgDefaultDeviceLimit":   "101",
		"awgReconcileIntervalSec": "4",
		"awgStatsIntervalSec":     "9",
		"awgPublicEndpoint":       "missing-port",
		"awgSubnet":               "2001:db8::/64",
	} {
		t.Run(key, func(t *testing.T) {
			values := validAWGSettingValues()
			values[key] = value
			if _, err := parseAWGSettings(values); err == nil {
				t.Fatalf("invalid %s=%q was accepted", key, value)
			}
		})
	}
}

func TestValidateAWGManagedEndpoint(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	settings, err := parseAWGSettings(validAWGSettingValues())
	if err != nil {
		t.Fatal(err)
	}
	options := map[string]any{
		"address":     []string{"10.77.0.1/16"},
		"private_key": base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"listen_port": 51820,
		"peers":       []any{},
		"amnezia": map[string]any{
			"jc": 3, "jmin": 10, "jmax": 20,
			"s1": 15, "s2": 18, "s3": 12, "s4": 8,
			"h1": "1000-1099", "h2": "2000-2099", "h3": "3000-3099", "h4": "4000-4099",
			"i1": "<b 0x01020304><r 8>",
		},
	}
	raw, err := json.Marshal(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Endpoint{Type: "wireguard", Tag: settings.EndpointTag, Options: raw}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ValidateAWGManagedEndpoint(db, settings); err != nil {
		t.Fatalf("valid endpoint rejected: %v", err)
	}
}

func TestValidateAWGManagedEndpointDoesNotLeakPrivateKey(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	settings, err := parseAWGSettings(validAWGSettingValues())
	if err != nil {
		t.Fatal(err)
	}
	secret := "not-a-valid-private-key" // #nosec G101 -- malformed test input, not a credential.
	raw, err := json.Marshal(map[string]any{
		"address": []string{"10.77.0.1/16"}, "private_key": secret, "listen_port": 51820,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Endpoint{Type: "wireguard", Tag: settings.EndpointTag, Options: raw}).Error; err != nil {
		t.Fatal(err)
	}
	err = ValidateAWGManagedEndpoint(db, settings)
	if err == nil {
		t.Fatal("invalid private key was accepted")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("validation error leaked private key: %v", err)
	}
}
