package capabilities

import (
	"reflect"
	"sort"
	"testing"
)

// These frozen literals are an exact copy of the hand-maintained lists that lived
// in service/inbounds.go (userJSONField, allowedUserJSONFields) and util/genLink.go
// (InboundTypeWithLink) BEFORE the manifest was introduced. The tests below prove
// the manifest derivation reproduces them, i.e. Phase 0 changes no behaviour. When
// a later phase deliberately changes one of these (e.g. Phase 2 adds mtproxy to the
// link set), update the corresponding golden here in the same change, with a note.

// legacyUserJSONField — frozen copy of service/inbounds.go userJSONField.
var legacyUserJSONField = map[string]string{
	"mixed":         "mixed",
	"socks":         "socks",
	"http":          "http",
	"shadowsocks":   "shadowsocks",
	"shadowsocks16": "shadowsocks",
	"vmess":         "vmess",
	"trojan":        "trojan",
	"naive":         "naive",
	"hysteria":      "hysteria",
	"shadowtls":     "shadowtls",
	"tuic":          "tuic",
	"hysteria2":     "hysteria2",
	"vless":         "vless",
	"anytls":        "anytls",
	"mieru":         "mieru",
	"trusttunnel":   "trusttunnel",
	"ssh":           "ssh",
	"mtproxy":       "mtproxy",
}

// legacyAllowedUserJSONFields — frozen copy of service/inbounds.go allowedUserJSONFields.
var legacyAllowedUserJSONFields = map[string]struct{}{
	"mixed":       {},
	"socks":       {},
	"http":        {},
	"shadowsocks": {},
	"vmess":       {},
	"trojan":      {},
	"naive":       {},
	"hysteria":    {},
	"shadowtls":   {},
	"tuic":        {},
	"hysteria2":   {},
	"vless":       {},
	"anytls":      {},
	"mieru":       {},
	"trusttunnel": {},
	"ssh":         {},
	"mtproxy":     {},
}

// legacyInboundTypeWithLink — the URI link set plus mtproxy. Phase 2 added mtproxy
// (clientDelivery=telegram -> a tg:// link), so it now joins the link set; the
// other 12 are the original URI types frozen from util/genLink.go.
var legacyInboundTypeWithLink = []string{
	"socks", "http", "mixed", "shadowsocks", "naive", "hysteria",
	"hysteria2", "anytls", "tuic", "vless", "trojan", "vmess",
	"mtproxy",
}

func TestUserJSONFieldsMatchesLegacy(t *testing.T) {
	got := UserJSONFields()
	if !reflect.DeepEqual(got, legacyUserJSONField) {
		t.Fatalf("derived userJSONField drifted from legacy.\n got: %v\nwant: %v", got, legacyUserJSONField)
	}
}

func TestAllowedUserJSONFieldsMatchesLegacy(t *testing.T) {
	got := AllowedUserJSONFields()
	if !reflect.DeepEqual(got, legacyAllowedUserJSONFields) {
		t.Fatalf("derived allowedUserJSONFields drifted from legacy.\n got: %v\nwant: %v", got, legacyAllowedUserJSONFields)
	}
}

func TestInboundTypesWithLinkMatchesLegacySet(t *testing.T) {
	got := InboundTypesWithLink()
	// Order is semantically irrelevant (consumers use it as a SQL IN set / test
	// iteration), so compare as sets while still guarding membership exactly.
	if !sameStringSet(got, legacyInboundTypeWithLink) {
		t.Fatalf("derived InboundTypeWithLink drifted from legacy.\n got: %v\nwant: %v", got, legacyInboundTypeWithLink)
	}
	if dups := duplicates(got); len(dups) > 0 {
		t.Fatalf("InboundTypesWithLink has duplicates: %v", dups)
	}
}

// TestAllowedFieldsAreExactlyUserFieldValues locks the defence-in-depth property:
// the SQL field allow-list is precisely the set of declared user-field values.
func TestAllowedFieldsAreExactlyUserFieldValues(t *testing.T) {
	allowed := AllowedUserJSONFields()
	for _, field := range UserJSONFields() {
		if _, ok := allowed[field]; !ok {
			t.Fatalf("user field %q is not in the allowed set", field)
		}
	}
	for field := range allowed {
		found := false
		for _, v := range UserJSONFields() {
			if v == field {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("allowed field %q has no backing user field", field)
		}
	}
}

// TestSkipOutJSONTypesMatchesLegacy locks FillOutJson's early-return set
// (formerly the literal switch case in util/outJson.go).
func TestSkipOutJSONTypesMatchesLegacy(t *testing.T) {
	want := map[string]struct{}{"direct": {}, "tun": {}, "redirect": {}, "tproxy": {}}
	if got := SkipOutJSONTypes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SkipOutJSONTypes drifted.\n got: %v\nwant: %v", got, want)
	}
}

// TestOutJSONBuildersMatchesLegacy locks the FillOutJson dispatch: which inbound
// types run a named builder ("base" no-op or a protocol builder) vs an empty
// builder (wipe). This is exactly the legacy second switch in util/outJson.go.
func TestOutJSONBuildersMatchesLegacy(t *testing.T) {
	want := map[string]string{
		// no-op "base" (legacy: case "http","socks","mixed","anytls")
		"socks": "base", "http": "base", "mixed": "base", "anytls": "base",
		// protocol builders (legacy: dedicated cases)
		"shadowsocks": "shadowsocks", "shadowtls": "shadowtls",
		"hysteria": "hysteria", "hysteria2": "hysteria2", "tuic": "tuic",
		"vless": "vless", "trojan": "trojan", "vmess": "vmess", "naive": "naive",
		// Phase 1: extended protocols now run dedicated JSON builders.
		"mieru": "mieru", "sudoku": "sudoku", "trusttunnel": "trusttunnel", "ssh": "ssh",
		// mtproxy stays wiped (no sing-box outbound; Telegram-only, Phase 2).
		"mtproxy": "",
		// early-return types still appear with empty builder (never reached)
		"direct": "", "tun": "", "redirect": "", "tproxy": "",
	}
	if got := OutJSONBuilders(); !reflect.DeepEqual(got, want) {
		t.Fatalf("OutJSONBuilders drifted.\n got: %v\nwant: %v", got, want)
	}
}

func TestManifestParsesAndValidates(t *testing.T) {
	// init() already ran validate(); re-run explicitly so a regression surfaces here.
	if err := validate(); err != nil {
		t.Fatalf("manifest failed validation: %v", err)
	}
	if len(Inbounds()) == 0 {
		t.Fatal("no inbound capabilities loaded")
	}
}

func sameStringSet(a, b []string) bool {
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	return reflect.DeepEqual(as, bs)
}

func duplicates(in []string) []string {
	seen := map[string]int{}
	for _, s := range in {
		seen[s]++
	}
	var dups []string
	for s, n := range seen {
		if n > 1 {
			dups = append(dups, s)
		}
	}
	return dups
}
