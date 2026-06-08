package service

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestPrepareClientSubSecretGeneratesUUIDV4(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{
		Enable:   true,
		Name:     "alice",
		Inbounds: []byte("[]"),
		Links:    []byte("[]"),
	}

	if err := (&ClientService{}).prepareClientSubSecret(database.GetDB(), &client, false); err != nil {
		t.Fatal(err)
	}
	if !uuidV4Pattern.MatchString(client.SubSecret) {
		t.Fatalf("sub secret is not uuid-v4: %q", client.SubSecret)
	}
}

func TestRotateSubSecretChangesExistingClientSecret(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{
		Enable:    true,
		Name:      "alice",
		SubSecret: "old-secret",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	name, err := (&ClientService{}).RotateSubSecret(strconv.FormatUint(uint64(client.Id), 10))
	if err != nil {
		t.Fatal(err)
	}
	if name != "alice" {
		t.Fatalf("unexpected client name: %s", name)
	}

	var stored model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubSecret == "" || stored.SubSecret == "old-secret" {
		t.Fatalf("sub secret was not rotated: %#v", stored)
	}
	if !uuidV4Pattern.MatchString(stored.SubSecret) {
		t.Fatalf("rotated sub secret is not uuid-v4: %q", stored.SubSecret)
	}
}
