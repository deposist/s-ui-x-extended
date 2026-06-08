package service

import (
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x/database/model"
)

// TestClientChangeNameJSONProducesValidChangesFeed pins the M5 fix: a client
// name containing JSON metacharacters must still produce a valid Changes.Obj
// that survives json.Marshal of the whole changes feed (the path CheckChanges
// uses). Before the fix, DepleteJob/ResetClients built Obj by raw string
// concatenation, so a name like `evil"name` made the entire CheckChanges
// response fail to serialize for every admin.
func TestClientChangeNameJSONProducesValidChangesFeed(t *testing.T) {
	names := []string{
		`evil"name`,
		`back\slash`,
		"tab\tchar",
		"line\nbreak",
		`"`,
		``,
		"normal-name",
		`"; DROP--`,
	}
	for _, name := range names {
		obj := clientChangeNameJSON(name)
		if !json.Valid(obj) {
			t.Fatalf("clientChangeNameJSON(%q) is not valid JSON: %s", name, obj)
		}
		var roundTrip string
		if err := json.Unmarshal(obj, &roundTrip); err != nil {
			t.Fatalf("clientChangeNameJSON(%q) does not unmarshal: %v", name, err)
		}
		if roundTrip != name {
			t.Fatalf("round-trip mismatch for %q: got %q", name, roundTrip)
		}
		// The real failure mode: CheckChanges -> c.JSON -> json.Marshal([]Changes).
		changes := []model.Changes{{Key: "clients", Action: "disable", Obj: obj}}
		if _, err := json.Marshal(changes); err != nil {
			t.Fatalf("changes feed with name %q failed to marshal: %v", name, err)
		}
	}
}

// TestRawConcatNameBreaksChangesFeed documents the exact bug the fix removes:
// the old `"\"" + name + "\""` concatenation yields invalid JSON for a quoted
// name, which makes json.Marshal of the changes feed fail.
func TestRawConcatNameBreaksChangesFeed(t *testing.T) {
	bad := json.RawMessage("\"" + `evil"name` + "\"")
	changes := []model.Changes{{Key: "clients", Action: "disable", Obj: bad}}
	if _, err := json.Marshal(changes); err == nil {
		t.Fatal("expected the raw-concatenation name to break json.Marshal of the changes feed")
	}
}
