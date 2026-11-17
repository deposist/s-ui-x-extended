package service

import (
	"encoding/json"
	"testing"
)

func TestConfigSaveRejectsInvalidListenerPortsWithoutPersisting(t *testing.T) {
	for _, key := range []string{"webPort", "subPort"} {
		for _, value := range []string{"junk", "0", "-1", "65536"} {
			t.Run(key+"/"+value, func(t *testing.T) {
				initSettingTestDB(t)
				settings := &SettingService{}
				before, err := settings.getString(key)
				if err != nil {
					t.Fatal(err)
				}
				payload, _ := json.Marshal(map[string]string{key: value})
				config := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
				if _, err := config.Save("settings", "set", payload, "", "admin", "example.com"); err == nil {
					t.Fatal("invalid listener port was accepted")
				}
				after, err := settings.getString(key)
				if err != nil || after != before {
					t.Fatalf("invalid port changed persisted settings: before=%q after=%q err=%v", before, after, err)
				}
			})
		}
	}
}

func TestConfigSaveAcceptsListenerPortBoundaries(t *testing.T) {
	initSettingTestDB(t)
	config := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	for _, key := range []string{"webPort", "subPort"} {
		for _, value := range []string{"1", "65535"} {
			payload, _ := json.Marshal(map[string]string{key: value})
			if _, err := config.Save("settings", "set", payload, "", "admin", "example.com"); err != nil {
				t.Fatalf("%s=%s: %v", key, value, err)
			}
		}
	}
}
