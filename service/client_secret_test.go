package service

import (
	"regexp"
	"strconv"
	"sync"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
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

func TestS6F10_PrepareClientSubSecretConcurrentEditConvergesOnPersistedSecret(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{
		Enable:    true,
		Name:      "alice",
		Inbounds:  []byte("[]"),
		Links:     []byte("[]"),
		SubSecret: "",
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}

	const goroutines = 100
	var wg sync.WaitGroup
	start := make(chan struct{})
	secrets := make(chan string, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c := model.Client{Id: client.Id}
			if err := (&ClientService{}).prepareClientSubSecret(database.GetDB(), &c, true); err != nil {
				t.Errorf("prepareClientSubSecret: %v", err)
				return
			}
			secrets <- c.SubSecret
		}()
	}
	close(start)
	wg.Wait()
	close(secrets)

	var stored model.Client
	if err := database.GetDB().Where("id = ?", client.Id).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubSecret == "" {
		t.Fatal("expected persisted sub_secret to be non-empty")
	}
	if !uuidV4Pattern.MatchString(stored.SubSecret) {
		t.Fatalf("persisted sub secret is not uuid-v4: %q", stored.SubSecret)
	}

	seen := map[string]int{}
	for secret := range secrets {
		if secret == "" {
			t.Fatal("caller returned an empty sub_secret")
		}
		if secret != stored.SubSecret {
			t.Fatalf("caller returned unsaved sub_secret %q, persisted %q", secret, stored.SubSecret)
		}
		seen[secret]++
	}
	if len(seen) != 1 || seen[stored.SubSecret] != goroutines {
		t.Fatalf("expected all callers to converge on persisted secret %q, got %#v", stored.SubSecret, seen)
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
