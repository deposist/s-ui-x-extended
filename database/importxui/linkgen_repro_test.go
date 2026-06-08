package importxui

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
)

// TestRepro_MigratedClientLinks reproduces the reported bug: after a 3x-ui
// migration the subscription/links show nothing for the imported inbounds.
//
//	IMPORT_XUI_REAL_DB="C:\\CheckErrorS-ui\\x-ui (6).db" go test ./database/importxui/ -run Repro_MigratedClientLinks -v
func TestRepro_MigratedClientLinks(t *testing.T) {
	path := os.Getenv("IMPORT_XUI_REAL_DB")
	if path == "" {
		t.Skip("set IMPORT_XUI_REAL_DB to a real x-ui.db to run this test")
	}
	initCompatDest(t)

	plan, err := Plan(path, PlanOptions{
		Strategy:        StrategyMerge,
		IncludeSettings: true,
		IncludeHistory:  false,
		IncludeRouting:  false,
		AdminMode:       AdminModeSkip,
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if _, err := Apply(path, *plan, ApplyOptions{Hostname: "panel.example.com"}); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	db := database.GetDB()

	var clients []model.Client
	if err := db.Model(model.Client{}).Find(&clients).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("imported %d clients", len(clients))

	emptyLinks := 0
	trojanChecked := false
	for _, c := range clients {
		var stored []map[string]string
		if len(c.Links) > 0 {
			if err := json.Unmarshal(c.Links, &stored); err != nil {
				t.Fatalf("client %q has unparseable Links: %v", c.Name, err)
			}
		}
		if len(stored) == 0 {
			emptyLinks++
			t.Logf("client=%-10s STORED_LINKS=0", c.Name)
			continue
		}
		for _, l := range stored {
			// Every stored link must bake the supplied hostname, never an
			// empty host (the broken-link regression).
			if !strings.Contains(l["uri"], "panel.example.com") {
				t.Errorf("client %q link missing hostname: %s", c.Name, l["uri"])
			}
			if strings.HasPrefix(l["uri"], "trojan://") && strings.Contains(l["uri"], "inbound-12223") {
				trojanChecked = true
				t.Logf("trojan-12223 link OK: %s", l["uri"])
			}
		}
	}
	if emptyLinks > 0 {
		t.Errorf("%d/%d clients still have empty stored Links after import with a hostname", emptyLinks, len(clients))
	}
	if !trojanChecked {
		t.Errorf("no client produced a stored trojan link for inbound-12223")
	}
}
