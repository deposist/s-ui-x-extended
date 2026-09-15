package core

import (
	"sort"
	"testing"

	"github.com/deposist/s-ui-x-extended/core/capabilities"
)

// TestRegistryTypesAreManifestDeclared is the type-level drift guard: every
// protocol type the pinned core registers must be accounted for in the
// capabilities manifest (protocols.json) — either as a declared entry, a
// documented alias, or a reviewed allowlist entry with a concrete reason.
//
// Why this exists: the field-level coverage gate (option_coverage_test.go)
// only checks structs listed in its explicit covMappings, so a brand-new core
// type would silently escape BOTH gates. This test closes that hole: adding a
// type to the fork's registries fails here until the panel makes a deliberate
// decision (expose it in the manifest, or allowlist it with a reason).
//
// The reverse direction is checked too: every manifest entry that is neither
// build-tagged nor an alias/panel-only row must correspond to a registered
// core type, so stale/typo'd manifest rows also fail.

// registryAliases maps a core-registered type to the manifest type that covers
// it, where the panel deliberately uses a different name.
var registryAliases = map[string]string{
	// The panel exposes the native core failover inbound as "core-failover";
	// the bare "failover" name is the panel-managed priority group assembled as
	// a selector (see the manifest groups section).
	"failover": "core-failover",
	// The manifest groups both core VPN endpoint types under one "vpn" row.
	"vpn-client": "vpn",
	"vpn-server": "vpn",
}

// registryAllowlist covers core types the panel deliberately does NOT expose.
// Every entry needs a concrete reason; a missing implementation is not one
// (it must be fixed or explicitly deferred with a tracking note).
var registryAllowlist = map[string]string{
	// Deprecated error stubs: registered so old configs fail with a clear
	// message instead of "unknown type". Must never be panel-exposed.
	"shadowsocksr": "deprecated error stub (removed in sing-box 1.6.0); registered only to reject old configs with a clear message",
	"wireguard":    "deprecated error stub (removed in sing-box 1.13.0); WireGuard is endpoint-only; the stub rejects outbound usage with a migration hint",
	// Build-tagged tunnel/USB services registered as stubs in the default
	// build. Not yet panel-exposed; adding an editor means adding a manifest
	// entry with the matching buildTag and removing the allowlist row.
	"cloudflared":  "with_cloudflared tunnel daemon inbound; no panel editor yet — expose via manifest when the editor lands",
	"usbip-client": "with_usbip service; no panel editor yet — expose via manifest when the editor lands",
	"usbip-server": "with_usbip service; no panel editor yet — expose via manifest when the editor lands",
	// The core registers snell in both directions; the stage-1 matrix exposes
	// the snell OUTBOUND only (declared in the manifest). The inbound side is
	// deliberately not panel-exposed.
	"snell": "core registers snell inbound+outbound; panel exposes the outbound only (manifest row); inbound not exposed by design",
}

// manifestOnlyTypes are manifest rows with no 1:1 core registry type: aliases
// and panel-managed composites.
var manifestOnlyTypes = map[string]string{
	"shadowsocks16": "alias:true — method alias of shadowsocks; backend user-field lookup only, never an independent inbound type",
}

func TestRegistryTypesAreManifestDeclared(t *testing.T) {
	declared := map[string]map[string]bool{
		"in":  {},
		"out": {},
		"ep":  {},
		"svc": {},
	}
	for _, in := range capabilities.Inbounds() {
		declared["in"][in.Type] = true
	}
	for _, out := range capabilities.Outbounds() {
		declared["out"][out.Type] = true
	}
	// Group modes are core outbounds (selector/urltest/fallback) or panel
	// composites assembled as one (failover -> selector).
	for _, g := range capabilities.Groups() {
		declared["out"][g.Type] = true
	}
	for _, ep := range capabilities.Endpoints() {
		declared["ep"][ep.Type] = true
	}
	for _, svc := range capabilities.Services() {
		declared["svc"][svc.Type] = true
	}

	// Manifest types covered through an alias (the core registers a different
	// name that maps onto them).
	aliasTargets := map[string]bool{}
	for _, manifest := range registryAliases {
		aliasTargets[manifest] = true
	}

	registries := []struct {
		name    string
		types   []string
		declare map[string]bool
	}{
		{"inbound", InboundRegistry().OptionTypes(), declared["in"]},
		{"outbound", OutboundRegistry().OptionTypes(), declared["out"]},
		{"endpoint", EndpointRegistry().OptionTypes(), declared["ep"]},
		{"service", ServiceRegistry().OptionTypes(), declared["svc"]},
	}

	registered := map[string]bool{}
	for _, reg := range registries {
		for _, ty := range reg.types {
			registered[ty] = true
			if reg.declare[ty] {
				continue
			}
			if manifest, ok := registryAliases[ty]; ok {
				if !reg.declare[manifest] {
					t.Errorf("%s registry type %q aliases to manifest type %q which is not declared", reg.name, ty, manifest)
				}
				continue
			}
			if _, ok := registryAllowlist[ty]; ok {
				continue
			}
			t.Errorf("%s registry type %q is not declared in the capabilities manifest and not allowlisted: a new core type would silently escape coverage — add it to core/capabilities/protocols.json (with an editor) or to registryAllowlist with a concrete reason", reg.name, ty)
		}
	}

	// Stale allowlist entries are dead weight and hide real decisions.
	for ty := range registryAllowlist {
		if !registered[ty] {
			t.Errorf("allowlist entry %q is not registered by the core in this build; remove it or fix the reason", ty)
		}
	}

	// Reverse direction: manifest rows without a build tag and without an
	// alias/panel-only justification must be registered core types. Build-tagged
	// rows may legitimately be stubs in the default test build, so they are
	// skipped here (the stub registrations still surface in the forward check
	// when compiled in).
	tagged := map[string]bool{}
	for _, in := range capabilities.Inbounds() {
		if in.BuildTag != "" {
			tagged[in.Type] = true
		}
	}
	for _, out := range capabilities.Outbounds() {
		if out.BuildTag != "" {
			tagged[out.Type] = true
		}
	}
	for _, ep := range capabilities.Endpoints() {
		if ep.BuildTag != "" {
			tagged[ep.Type] = true
		}
	}
	for _, svc := range capabilities.Services() {
		if svc.BuildTag != "" {
			tagged[svc.Type] = true
		}
	}
	aliasSources := map[string]bool{}
	for core := range registryAliases {
		aliasSources[core] = true
	}
	var undeclared []string
	for _, reg := range registries {
		for ty := range reg.declare {
			if tagged[ty] {
				continue
			}
			if _, ok := manifestOnlyTypes[ty]; ok {
				continue
			}
			if aliasTargets[ty] {
				continue // covered by aliased core types (vpn, core-failover)
			}
			if registered[ty] || aliasSources[ty] {
				continue
			}
			undeclared = append(undeclared, reg.name+":"+ty)
		}
	}
	if len(undeclared) > 0 {
		sort.Strings(undeclared)
		t.Errorf("manifest declares types the core does not register (stale rows or typos): %v", undeclared)
	}
}
