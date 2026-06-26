package capabilities

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
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
	want := map[string]struct{}{"direct": {}, "tun": {}, "redirect": {}, "tproxy": {}, "bond": {}, "core-failover": {}}
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
		// native core inbound types with no client delivery
		"bond": "", "core-failover": "",
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
	if len(Groups()) == 0 {
		t.Fatal("no group capabilities loaded")
	}
}

func TestGroupCapabilitiesDocumentPanelFailoverBoundary(t *testing.T) {
	groups := Groups()
	byType := make(map[string]GroupCapability, len(groups))
	for _, group := range groups {
		byType[group.Type] = group
	}
	for _, groupType := range []string{"selector", "urltest", "fallback", "failover"} {
		if _, ok := byType[groupType]; !ok {
			t.Fatalf("missing group capability %q", groupType)
		}
	}

	failover := byType["failover"]
	if !failover.PanelManaged {
		t.Fatal("panel failover must be marked panelManaged")
	}
	if failover.AssembledAs != "selector" {
		t.Fatalf("panel failover assembledAs = %q, want selector", failover.AssembledAs)
	}
	if failover.SessionRecovery {
		t.Fatal("panel failover must not claim generic session recovery")
	}
	if failover.CoreType != "" {
		t.Fatalf("panel failover coreType = %q, want empty because it is assembled by the panel", failover.CoreType)
	}
	if failover.Notes == "" {
		t.Fatal("panel failover should carry an operator-facing boundary note")
	}

	for _, groupType := range []string{"selector", "urltest", "fallback"} {
		group := byType[groupType]
		if group.PanelManaged {
			t.Fatalf("core-backed group %q must not be marked panelManaged", groupType)
		}
		if group.CoreType != groupType {
			t.Fatalf("core-backed group %q coreType = %q, want same type", groupType, group.CoreType)
		}
		if group.SessionRecovery {
			t.Fatalf("core-backed group %q must not claim session recovery", groupType)
		}
	}
}

func TestReleaseBuildTagsCoverManifestProtocolCapabilities(t *testing.T) {
	manifestTags := manifestProtocolBuildTagSet()
	if len(manifestTags) == 0 {
		t.Fatal("manifest exposes no build-tagged protocol capabilities")
	}

	releaseTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, ".github/workflows/release.yml"), `(?m)^\s*RELEASE_BUILD_TAGS:\s*"([^"]+)"\s*$`))
	for _, extra := range parseQuotedAssignmentAll(t, readRepoFile(t, ".github/workflows/release.yml"), `(?m)BUILD_TAGS="\$\{BUILD_TAGS\},([^"]+)"`) {
		addTags(releaseTags, extra)
	}
	assertContainsAllTags(t, "release.yml effective build tags", releaseTags, manifestTags)

	windowsTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, ".github/workflows/windows.yml"), `(?m)^\s*TAGS:\s*"([^"]+)"\s*$`))
	assertContainsAllTags(t, "windows.yml TAGS", windowsTags, manifestTags)

	batTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, "windows/build-windows.bat"), `(?m)^set BUILD_TAGS=([^\r\n]+)`))
	assertContainsAllTags(t, "windows/build-windows.bat BUILD_TAGS", batTags, manifestTags)

	psTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, "windows/build-windows.ps1"), `(?m)^\$buildTags = "([^"]+)"\s*$`))
	assertContainsAllTags(t, "windows/build-windows.ps1 buildTags", psTags, manifestTags)

	buildScriptTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, "build.sh"), `(?m)^BUILD_TAGS="([^"]+)"\s*$`))
	assertContainsAllTags(t, "build.sh BUILD_TAGS", buildScriptTags, manifestTags)

	dockerfileTags := parseTagSet(t, parseQuotedAssignment(t, readRepoFile(t, "Dockerfile"), `(?m)-tags "([^"]+)"`))
	assertContainsAllTags(t, "Dockerfile build tags", dockerfileTags, manifestTags)
}

func manifestProtocolBuildTagSet() map[string]struct{} {
	tags := map[string]struct{}{}
	for _, in := range Inbounds() {
		if in.BuildTag != "" {
			tags[in.BuildTag] = struct{}{}
		}
	}
	for _, out := range Outbounds() {
		if out.BuildTag != "" {
			tags[out.BuildTag] = struct{}{}
		}
	}
	for _, endpoint := range Endpoints() {
		if endpoint.BuildTag != "" {
			tags[endpoint.BuildTag] = struct{}{}
		}
	}
	for _, provider := range Providers() {
		if provider.BuildTag != "" {
			tags[provider.BuildTag] = struct{}{}
		}
	}
	return tags
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func parseQuotedAssignment(t *testing.T, text, pattern string) string {
	t.Helper()
	matches := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if matches == nil {
		t.Fatalf("pattern %q not found", pattern)
	}
	return matches[1]
}

func parseQuotedAssignmentAll(t *testing.T, text, pattern string) []string {
	t.Helper()
	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}

func parseTagSet(t *testing.T, raw string) map[string]struct{} {
	t.Helper()
	tags := map[string]struct{}{}
	addTags(tags, raw)
	return tags
}

func addTags(tags map[string]struct{}, raw string) {
	for _, tag := range strings.Split(raw, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags[tag] = struct{}{}
		}
	}
}

func assertContainsAllTags(t *testing.T, label string, got, want map[string]struct{}) {
	t.Helper()
	var missing []string
	for tag := range want {
		if _, ok := got[tag]; !ok {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("%s missing manifest build tags: %v", label, missing)
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
