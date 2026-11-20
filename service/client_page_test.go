package service

import (
	"slices"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestClientListPageFiltersAndSortsAcrossWholeCatalog(t *testing.T) {
	initServicePerfDB(t)
	clients := []model.Client{
		{Name: "Zulu", Desc: "ПРИВЕТ", Group: "team", Up: 3, Enable: true, Inbounds: []byte("[]")},
		{Name: "Bravo", Desc: "Привет", Group: "team", Up: 12, Enable: true, Inbounds: []byte("[]")},
		{Name: "Alpha", Desc: "привет", Group: "team", Up: 7, Enable: true, Inbounds: []byte("[]")},
		{Name: "Offline[1]", Group: "", Enable: false, Inbounds: []byte("[]")},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}
	service := &ClientService{}
	query := ClientListQuery{
		Page: 2, Size: 1, Group: "team", Search: "привет", State: "online",
		Sort: "name", Fold: true, Online: []string{"Zulu", "Alpha", "deleted"},
	}
	got, err := service.ListPage(query)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 4 || got.Filtered != 2 || got.Online != 2 || got.Page != 2 ||
		len(got.Clients) != 1 || got.Clients[0].Name != "Zulu" || !slices.Equal(got.Groups, []string{"", "team"}) {
		t.Fatalf("bad page metadata or row: %#v", got)
	}
	query.Group, query.Search, query.State = "", "", ""
	got, err = service.ListPage(query)
	if err != nil || got.Page != 1 || len(got.Clients) != 1 || got.Clients[0].Name != "Offline[1]" {
		t.Fatalf("empty group or page clamp failed: %#v / %v", got, err)
	}
	query.Group, query.Search, query.Fold = "-", "[", false
	got, err = service.ListPage(query)
	if err != nil || got.Filtered != 1 || len(got.Clients) != 1 || got.Clients[0].Name != "Offline[1]" {
		t.Fatalf("literal punctuation filter failed: %#v / %v", got, err)
	}
}
