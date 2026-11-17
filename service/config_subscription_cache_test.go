package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestClientSubscriptionCacheIDsLoadsOmittedSecretsFromDatabase(t *testing.T) {
	initSettingTestDB(t)
	clients := []model.Client{
		{Name: "cache-a", SubSecret: "secret-a", Inbounds: json.RawMessage(`[]`)},
		{Name: "cache-b", SubSecret: "secret-b", Inbounds: json.RawMessage(`[]`)},
	}
	for i := range clients {
		if err := database.GetDB().Create(&clients[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	payload, _ := json.Marshal([]model.Client{{Id: clients[0].Id}, {Id: clients[1].Id}})
	got := clientSubscriptionCacheIDs("clients", payload)
	if !reflect.DeepEqual(got, []string{"secret-a", "secret-b"}) {
		t.Fatalf("cache IDs=%v, want both persisted secrets", got)
	}
}

func TestClientSubscriptionCacheIDsIncludesMixedBulkSecrets(t *testing.T) {
	initSettingTestDB(t)
	clients := []model.Client{
		{Name: "mixed-a", SubSecret: "secret-a", Inbounds: json.RawMessage(`[]`)},
		{Name: "mixed-b", SubSecret: "secret-b", Inbounds: json.RawMessage(`[]`)},
	}
	for i := range clients {
		if err := database.GetDB().Create(&clients[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	payload, _ := json.Marshal([]model.Client{{Id: clients[0].Id, SubSecret: "secret-a"}, {Id: clients[1].Id}})
	got := clientSubscriptionCacheIDs("clients", payload)
	if !reflect.DeepEqual(got, []string{"secret-a", "secret-b"}) {
		t.Fatalf("cache IDs=%v, want both provided and persisted secrets", got)
	}
}

func TestClientSubscriptionCacheIDsIncludesPreviousSecretAfterRotation(t *testing.T) {
	initSettingTestDB(t)
	client := model.Client{Name: "rotate-cache", SubSecret: "old-secret", Inbounds: json.RawMessage(`[]`)}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(model.Client{Id: client.Id, SubSecret: "new-secret"})
	got := clientSubscriptionCacheIDs("clients", payload)
	if !reflect.DeepEqual(got, []string{"new-secret", "old-secret"}) {
		t.Fatalf("cache IDs=%v, want current and previous secrets", got)
	}
}
