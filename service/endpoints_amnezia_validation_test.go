package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func amneziaEndpointPayload(id uint, tag string, amnezia string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"id":%d,"type":"wireguard","tag":%q,"system":false,"address":["10.0.0.2/32"],"private_key":%q,"peers":[],"mtu":1408,"amnezia":%s}`,
		id, tag, testWireguardKey, amnezia))
}

// Saving an endpoint with an invalid Amnezia combination must be rejected
// outright (project decision: hard block, no escape hatch) because such a
// tunnel silently never comes up.
func TestSaveEndpointRejectsInvalidAmneziaParams(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))

	cases := []struct {
		name     string
		amnezia  string
		errChunk string
	}{
		{
			"reserved legacy headers",
			`{"jc":3,"jmin":10,"jmax":20,"h1":1,"h2":2,"h3":3,"h4":4}`,
			"reserved",
		},
		{
			"overlapping header ranges",
			`{"jc":3,"jmin":10,"jmax":20,"h1":"1000-2000","h2":"1500-2500","h3":5000,"h4":6000}`,
			"overlap",
		},
		{
			"jmin above jmax",
			`{"jc":3,"jmin":90,"jmax":40,"h1":1000,"h2":2000,"h3":3000,"h4":4000}`,
			"Jmin",
		},
		{
			"s1 plus 56 equals s2",
			`{"jc":3,"jmin":10,"jmax":20,"s1":15,"s2":71,"h1":1000,"h2":2000,"h3":3000,"h4":4000}`,
			"equal packet sizes",
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := amneziaEndpointPayload(0, fmt.Sprintf("awg-invalid-%d", i), tc.amnezia)
			_, err := configService.Save("endpoints", "new", payload, "", "admin", "example.com")
			if err == nil {
				t.Fatal("expected save to be rejected")
			}
			if !strings.Contains(err.Error(), tc.errChunk) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.errChunk)
			}
		})
	}
}

func TestSaveEndpointAcceptsValidAmneziaParams(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload := amneziaEndpointPayload(0, "awg-valid",
		`{"jc":4,"jmin":40,"jmax":90,"s1":15,"s2":20,"s3":12,"s4":8,"h1":"1000-1099","h2":2000,"h3":"3000-3099","h4":"4000-4099"}`)
	if _, err := configService.Save("endpoints", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("valid Amnezia endpoint rejected: %v", err)
	}
}

// Endpoints without an amnezia section (plain WireGuard) must keep saving.
func TestSaveEndpointWithoutAmneziaUnaffected(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	payload := json.RawMessage(fmt.Sprintf(
		`{"type":"wireguard","tag":"wg-plain-amnezia-check","system":false,"address":["10.0.0.2/32"],"private_key":%q,"peers":[],"mtu":1408}`,
		testWireguardKey))
	if _, err := configService.Save("endpoints", "new", payload, "", "admin", "example.com"); err != nil {
		t.Fatalf("plain WireGuard endpoint rejected: %v", err)
	}
}

// Editing an existing endpoint with invalid Amnezia params must also be
// blocked: old endpoints are untouched until first edit, at which point the
// combination has to be fixed.
func TestEditEndpointRejectsInvalidAmneziaParams(t *testing.T) {
	initSettingTestDB(t)
	configService := NewConfigServiceWithRuntime(NewRuntimeWithCoreProvider(nil))
	valid := amneziaEndpointPayload(0, "awg-edit-check",
		`{"jc":4,"jmin":40,"jmax":90,"h1":1000,"h2":2000,"h3":3000,"h4":4000}`)
	if _, err := configService.Save("endpoints", "new", valid, "", "admin", "example.com"); err != nil {
		t.Fatalf("setup save failed: %v", err)
	}
	var saved model.Endpoint
	if err := database.GetDB().Where("tag = ?", "awg-edit-check").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	invalid := amneziaEndpointPayload(saved.Id, "awg-edit-check",
		`{"jc":4,"jmin":40,"jmax":90,"h1":9999,"h2":9999,"h3":3000,"h4":4000}`)
	if _, err := configService.Save("endpoints", "edit", invalid, "", "admin", "example.com"); err == nil {
		t.Fatal("expected edit with overlapping headers to be rejected")
	}
}
