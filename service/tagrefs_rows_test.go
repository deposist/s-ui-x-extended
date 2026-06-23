package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestScanServiceRowsForInboundTag(t *testing.T) {
	rows := []model.Service{
		{Id: 1, Type: "ssm-api", Tag: "ssm", Options: json.RawMessage(`{"servers":{"/main":"ss-in","/extra":"other-in"}}`)},
		{Id: 2, Type: "ssm-api", Tag: "ssm-other", Options: json.RawMessage(`{"servers":{"/x":"unrelated"}}`)},
		{Id: 3, Type: "derp", Tag: "derp", Options: json.RawMessage(`{"servers":{"/y":"ss-in"}}`)},
		{Id: 4, Type: "ssm-api", Tag: "broken", Options: json.RawMessage(`{"servers":"not-a-map"`)},
	}

	refs := scanServiceRowsForInboundTag(rows, "ss-in")
	if len(refs) != 1 {
		t.Fatalf("refs = %+v, want exactly one match", refs)
	}
	if refs[0].Lazy {
		t.Fatal("ssm-api binding is eager, must not be marked lazy")
	}
	wantLocator := `ssm-api service "ssm" (servers["/main"])`
	if refs[0].Locator != wantLocator {
		t.Fatalf("locator = %q, want %q", refs[0].Locator, wantLocator)
	}

	if got := scanServiceRowsForInboundTag(rows, "missing"); len(got) != 0 {
		t.Fatalf("unreferenced tag produced refs: %+v", got)
	}
}

func TestSsmServiceIdsReferencingInbound(t *testing.T) {
	rows := []model.Service{
		{Id: 7, Type: "ssm-api", Tag: "a", Options: json.RawMessage(`{"servers":{"/1":"ss-in","/2":"ss-in"}}`)},
		{Id: 8, Type: "ssm-api", Tag: "b", Options: json.RawMessage(`{"servers":{"/3":"other"}}`)},
		{Id: 9, Type: "ssm-api", Tag: "c", Options: json.RawMessage(`{"servers":{"/4":"ss-in"}}`)},
	}

	ids := ssmServiceIdsReferencingInbound(rows, "ss-in")
	if !reflect.DeepEqual(ids, []uint{7, 9}) {
		t.Fatalf("ids = %v, want [7 9] (each service once)", ids)
	}
}

func TestFormatTagReferenceErrorEnumeratesAndHints(t *testing.T) {
	err := formatTagReferenceError("outbound", "proxy-a", []TagReference{
		{Kind: "selector", Locator: `selector "auto" (outbounds list)`},
		{Kind: "route rule", Locator: "route rule #3 (outbound)", Lazy: true},
	})
	msg := err.Error()
	for _, fragment := range []string{
		`outbound "proxy-a"`,
		`selector "auto" (outbounds list)`,
		"route rule #3 (outbound)",
		"point it to another outbound (e.g. direct)",
	} {
		if !strings.Contains(msg, fragment) {
			t.Fatalf("error %q does not contain %q", msg, fragment)
		}
	}
}

func TestScanOutboundRowsForTag(t *testing.T) {
	rows := []model.Outbound{
		{Id: 1, Type: "socks", Tag: "relay", Options: json.RawMessage(`{"server":"127.0.0.1","detour":"proxy-a"}`)},
		{Id: 2, Type: "selector", Tag: "auto", Options: json.RawMessage(`{"outbounds":["proxy-a","direct"],"default":"proxy-a"}`)},
		{Id: 3, Type: "urltest", Tag: "fastest", Options: json.RawMessage(`{"outbounds":["direct"]}`)},
		{Id: 4, Type: "socks", Tag: "self", Options: json.RawMessage(`{"detour":"proxy-a"}`)},
		{Id: 5, Type: "socks", Tag: "broken", Options: json.RawMessage(`{"detour":`)},
	}

	refs := scanOutboundRowsForTag(rows, "proxy-a", 4)
	wantLocators := map[string]bool{
		`outbound "relay" (detour)`:        false,
		`selector "auto" (outbounds list)`: false,
		`selector "auto" (default)`:        false,
	}
	for _, ref := range refs {
		if ref.Lazy {
			t.Fatalf("outbound row reference %q must be eager", ref.Locator)
		}
		if _, expected := wantLocators[ref.Locator]; !expected {
			t.Fatalf("unexpected reference %q (excluded row id=4 leaked or broken row blocked?)", ref.Locator)
		}
		wantLocators[ref.Locator] = true
	}
	for locator, seen := range wantLocators {
		if !seen {
			t.Fatalf("missing reference %q in %+v", locator, refs)
		}
	}
}

func TestScanEndpointAndServiceRowsForOutboundDetour(t *testing.T) {
	endpointRefs := scanEndpointRowsForTag([]model.Endpoint{
		{Id: 1, Type: "wireguard", Tag: "wg0", Options: json.RawMessage(`{"detour":"proxy-a"}`)},
		{Id: 2, Type: "wireguard", Tag: "wg-self", Options: json.RawMessage(`{"detour":"proxy-a"}`)},
	}, "proxy-a", 2)
	if len(endpointRefs) != 1 || endpointRefs[0].Locator != `endpoint "wg0" (detour)` {
		t.Fatalf("endpoint refs = %+v, want only wg0", endpointRefs)
	}

	serviceRefs := scanServiceRowsForOutboundDetour([]model.Service{
		{Id: 1, Type: "ccm", Tag: "ccm-svc", Options: json.RawMessage(`{"detour":"proxy-a"}`)},
		{Id: 2, Type: "ssm-api", Tag: "ssm", Options: json.RawMessage(`{"servers":{"/m":"ss-in"}}`)},
	}, "proxy-a")
	if len(serviceRefs) != 1 || serviceRefs[0].Locator != `ccm service "ccm-svc" (detour)` {
		t.Fatalf("service refs = %+v, want only ccm-svc", serviceRefs)
	}
}

func TestEagerTagReferencesFiltersLazy(t *testing.T) {
	refs := []TagReference{
		{Locator: "eager-1"},
		{Locator: "lazy-1", Lazy: true},
		{Locator: "eager-2"},
	}
	eager := eagerTagReferences(refs)
	if len(eager) != 2 || eager[0].Locator != "eager-1" || eager[1].Locator != "eager-2" {
		t.Fatalf("eager refs = %+v, want the two eager entries", eager)
	}
	if got := eagerTagReferences([]TagReference{{Locator: "l", Lazy: true}}); len(got) != 0 {
		t.Fatalf("lazy-only input must filter to empty, got %+v", got)
	}
}
