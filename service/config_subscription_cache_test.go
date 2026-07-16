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
