package service

import (
	"testing"
)

func findRef(refs []TagReference, locator string) *TagReference {
	for i := range refs {
		if refs[i].Locator == locator {
			return &refs[i]
		}
	}
	return nil
}

func TestScanConfigBlobForOutboundTagCoversAllReferenceSites(t *testing.T) {
	blob := []byte(`{
		"log": {"level": "info"},
		"dns": {"servers": [
			{"tag": "dns-remote", "address": "1.1.1.1", "detour": "proxy-a"},
			{"tag": "dns-local", "address": "local"}
		]},
		"ntp": {"enabled": true, "detour": "proxy-a"},
		"route": {
			"final": "proxy-a",
			"rules": [
				{"protocol": "dns", "outbound": "proxy-a"},
				{"type": "logical", "mode": "and", "rules": [{"port": 53, "outbound": "proxy-a"}]},
				{"network": "udp", "outbound": "other"}
			],
			"rule_set": [{"tag": "geoip", "type": "remote", "download_detour": "proxy-a"}]
		},
		"experimental": {
			"clash_api": {"external_ui_download_detour": "proxy-a"},
			"v2ray_api": {"stats": {"enabled": true, "outbounds": ["proxy-a"]}}
		}
	}`)

	refs, err := scanConfigBlobForOutboundTag(blob, "proxy-a")
	if err != nil {
		t.Fatal(err)
	}

	eagerLocators := []string{
		`dns server "dns-remote" (detour)`,
		"ntp (detour)",
		`rule_set "geoip" (download_detour)`,
		"clash_api (external_ui_download_detour)",
	}
	lazyLocators := []string{
		"route rule #0 (outbound)",
		"route rule #1 (outbound)",
		"route final",
	}
	for _, locator := range eagerLocators {
		ref := findRef(refs, locator)
		if ref == nil {
			t.Fatalf("missing eager reference %q in %+v", locator, refs)
		}
		if ref.Lazy {
			t.Fatalf("reference %q must be eager", locator)
		}
	}
	for _, locator := range lazyLocators {
		ref := findRef(refs, locator)
		if ref == nil {
			t.Fatalf("missing lazy reference %q in %+v", locator, refs)
		}
		if !ref.Lazy {
			t.Fatalf("reference %q must be lazy", locator)
		}
	}
	if len(refs) != len(eagerLocators)+len(lazyLocators) {
		t.Fatalf("unexpected extra references (v2ray stats must be ignored): %+v", refs)
	}
}

func TestScanConfigBlobForOutboundTagEdgeCases(t *testing.T) {
	if refs, err := scanConfigBlobForOutboundTag(nil, "x"); err != nil || len(refs) != 0 {
		t.Fatalf("missing blob must yield no refs, got %+v / %v", refs, err)
	}
	if refs, err := scanConfigBlobForOutboundTag([]byte(`{"dns":{}}`), "x"); err != nil || len(refs) != 0 {
		t.Fatalf("unreferenced tag must yield no refs, got %+v / %v", refs, err)
	}
	if _, err := scanConfigBlobForOutboundTag([]byte(`{"dns":`), "x"); err == nil {
		t.Fatal("malformed blob must be reported as an error")
	}
	// dns server without a tag falls back to its index in the locator.
	refs, err := scanConfigBlobForOutboundTag([]byte(`{"dns":{"servers":[{"address":"1.1.1.1","detour":"x"}]}}`), "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Locator != "dns server #0 (detour)" {
		t.Fatalf("untagged dns server locator = %+v, want dns server #0 (detour)", refs)
	}
}

