// Package capabilities is the single Go-side reader of core/capabilities/protocols.json,
// the embedded source-of-truth manifest for protocol capabilities. The manifest is
// also consumed by the frontend (scripts/gen-capabilities.cjs). Hand-maintained
// per-type lists that used to live in service/inbounds.go, util/genLink.go and
// util/outJson.go are now DERIVED here so they cannot drift apart.
//
// The manifest is embedded at build time, so every map/list returned below is a
// compile-time constant projection — in particular the SQL user-field allow-list
// (AllowedUserJSONFields) can only ever contain the manifest's declared, validated
// field identifiers.
package capabilities

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

//go:embed protocols.json
var manifestJSON []byte

// InboundCapability is one inbound type's row in the manifest. See protocols.json
// for field semantics. Only inbound entries carry the full field set.
type InboundCapability struct {
	Type           string            `json:"type"`
	HasUsers       bool              `json:"hasUsers"`
	UserField      string            `json:"userField"`
	Alias          bool              `json:"alias"`
	ClientDelivery string            `json:"clientDelivery"`
	LinkScheme     string            `json:"linkScheme"`
	OutJSONBuilder string            `json:"outJsonBuilder"`
	SkipOutJSON    bool              `json:"skipOutJson"`
	HasInData      bool              `json:"hasInData"`
	HasTLSTemplate bool              `json:"hasTlsTemplate"`
	MuxAvailable   bool              `json:"muxAvailable"`
	OnlyTLS        bool              `json:"onlyTls"`
	CredentialMap  map[string]string `json:"credentialMap"`
	UIEditor       string            `json:"uiEditor"`
	BuildTag       string            `json:"buildTag"`
	Notes          string            `json:"notes"`
}

// SimpleCapability is a light row for the non-inbound categories (outbounds /
// endpoints / services), used only to render the protocol matrix.
type SimpleCapability struct {
	Type     string `json:"type"`
	BuildTag string `json:"buildTag"`
	Notes    string `json:"notes"`
}

// GroupCapability describes panel/core outbound group modes. Some group modes
// are direct core types (selector/urltest/fallback), while panel-managed modes
// can assemble into a different core type. In particular, panel failover is a
// priority policy over a core selector and must not be presented as generic
// session recovery.
type GroupCapability struct {
	Type            string `json:"type"`
	CoreType        string `json:"coreType,omitempty"`
	AssembledAs     string `json:"assembledAs,omitempty"`
	PanelManaged    bool   `json:"panelManaged,omitempty"`
	SessionRecovery bool   `json:"sessionRecovery"`
	Notes           string `json:"notes"`
}

type manifest struct {
	Version   int                 `json:"version"`
	Inbounds  []InboundCapability `json:"inbounds"`
	Outbounds []SimpleCapability  `json:"outbounds"`
	Groups    []GroupCapability   `json:"groups"`
	Endpoints []SimpleCapability  `json:"endpoints"`
	Providers []SimpleCapability  `json:"providers"`
	Services  []SimpleCapability  `json:"services"`
}

var loaded manifest

// fieldIdent constrains user-field identifiers to plain tokens. AllowedUserJSONFields
// is interpolated into a JSON path inside a raw SQL string in
// service/inbounds.go:fetchUsersByCondition, so anything containing SQL/JSON-path
// metacharacters must never reach it. Validated at init for defence-in-depth.
var fieldIdent = regexp.MustCompile(`^[a-z0-9_]+$`)

var validClientDelivery = map[string]struct{}{
	"none": {}, "json": {}, "uri": {}, "telegram": {}, "broken": {},
}

func init() {
	if err := json.Unmarshal(manifestJSON, &loaded); err != nil {
		panic(fmt.Sprintf("capabilities: cannot parse protocols.json: %v", err))
	}
	if err := validate(); err != nil {
		panic("capabilities: " + err.Error())
	}
}

func validate() error {
	seen := map[string]struct{}{}
	for _, in := range loaded.Inbounds {
		if in.Type == "" {
			return fmt.Errorf("inbound entry with empty type")
		}
		if _, dup := seen[in.Type]; dup {
			return fmt.Errorf("duplicate inbound type %q", in.Type)
		}
		seen[in.Type] = struct{}{}

		if _, ok := validClientDelivery[in.ClientDelivery]; !ok {
			return fmt.Errorf("inbound %q has invalid clientDelivery %q", in.Type, in.ClientDelivery)
		}
		// Invariant: a type with users must declare the JSON field used to find them,
		// and that field must be a safe identifier (it reaches a raw SQL path).
		if in.HasUsers {
			if in.UserField == "" {
				return fmt.Errorf("inbound %q hasUsers but no userField", in.Type)
			}
			if !fieldIdent.MatchString(in.UserField) {
				return fmt.Errorf("inbound %q userField %q is not a safe identifier", in.Type, in.UserField)
			}
		} else if in.UserField != "" {
			return fmt.Errorf("inbound %q has userField but hasUsers is false", in.Type)
		}
		// Invariant: a URI/Telegram delivery type that is actually wired up must have a
		// link scheme. (A type may sit at clientDelivery=none/json/broken with no scheme.)
		if (in.ClientDelivery == "uri" || in.ClientDelivery == "telegram") && in.LinkScheme == "" {
			return fmt.Errorf("inbound %q has clientDelivery %q but no linkScheme", in.Type, in.ClientDelivery)
		}
		// Invariant: clientDelivery=none must not also claim a JSON out_json builder.
		if in.ClientDelivery == "none" && in.OutJSONBuilder != "" {
			return fmt.Errorf("inbound %q is clientDelivery=none but declares outJsonBuilder %q", in.Type, in.OutJSONBuilder)
		}
	}

	seenGroups := map[string]struct{}{}
	for _, group := range loaded.Groups {
		if group.Type == "" {
			return fmt.Errorf("group entry with empty type")
		}
		if _, dup := seenGroups[group.Type]; dup {
			return fmt.Errorf("duplicate group type %q", group.Type)
		}
		seenGroups[group.Type] = struct{}{}
		if group.CoreType == "" && group.AssembledAs == "" {
			return fmt.Errorf("group %q must declare coreType or assembledAs", group.Type)
		}
		if group.PanelManaged && group.AssembledAs == "" {
			return fmt.Errorf("panel-managed group %q must declare assembledAs", group.Type)
		}
		if group.Type == "failover" {
			if !group.PanelManaged {
				return fmt.Errorf("panel failover group must be panelManaged")
			}
			if group.AssembledAs != "selector" {
				return fmt.Errorf("panel failover group must assemble as selector")
			}
			if group.SessionRecovery {
				return fmt.Errorf("panel failover group must not claim session recovery")
			}
		}
	}
	return nil
}

// Inbounds returns the inbound capability rows in manifest order.
func Inbounds() []InboundCapability {
	out := make([]InboundCapability, len(loaded.Inbounds))
	copy(out, loaded.Inbounds)
	return out
}

// Outbounds, Endpoints, Providers and Services expose the light rows for the matrix generator.
func Outbounds() []SimpleCapability { return cloneSimple(loaded.Outbounds) }
func Endpoints() []SimpleCapability { return cloneSimple(loaded.Endpoints) }
func Providers() []SimpleCapability { return cloneSimple(loaded.Providers) }
func Services() []SimpleCapability  { return cloneSimple(loaded.Services) }

// Groups returns panel/core outbound group capability rows in manifest order.
func Groups() []GroupCapability {
	out := make([]GroupCapability, len(loaded.Groups))
	copy(out, loaded.Groups)
	return out
}

func cloneSimple(in []SimpleCapability) []SimpleCapability {
	out := make([]SimpleCapability, len(in))
	copy(out, in)
	return out
}

// UserJSONFields derives the inbound-type -> clients.config JSON field map
// (formerly the hand-written userJSONField in service/inbounds.go). Includes
// alias types (e.g. shadowsocks16). A fresh map is returned on every call so a
// caller can mutate its copy without affecting anyone else.
func UserJSONFields() map[string]string {
	m := map[string]string{}
	for _, in := range loaded.Inbounds {
		if in.HasUsers {
			m[in.Type] = in.UserField
		}
	}
	return m
}

// AllowedUserJSONFields derives the set of distinct JSON field values that may be
// embedded into the user-lookup SQL path (formerly the hand-written
// allowedUserJSONFields). Build-time constant by construction.
func AllowedUserJSONFields() map[string]struct{} {
	m := map[string]struct{}{}
	for _, in := range loaded.Inbounds {
		if in.HasUsers {
			m[in.UserField] = struct{}{}
		}
	}
	return m
}

// AssignableKeylessTypes returns inbound types that carry no per-user
// credential objects but ARE client-assignable because their delivery is
// the JSON subscription (clientDelivery == "json" && !hasUsers). Today this
// is exactly {sudoku}: keyless, yet a client must be assignable to the
// inbound to receive its outbound in the JSON subscription.
func AssignableKeylessTypes() map[string]struct{} {
	m := map[string]struct{}{}
	for _, in := range loaded.Inbounds {
		if !in.HasUsers && !in.Alias && in.ClientDelivery == "json" {
			m[in.Type] = struct{}{}
		}
	}
	return m
}

// InboundTypesWithLink derives the ordered list of inbound types that produce an
// external link (URI or Telegram), formerly util.InboundTypeWithLink. Order is
// manifest order; all consumers use it as a set (SQL IN, test iteration).
func InboundTypesWithLink() []string {
	var out []string
	for _, in := range loaded.Inbounds {
		if in.ClientDelivery == "uri" || in.ClientDelivery == "telegram" {
			out = append(out, in.Type)
		}
	}
	return out
}

// OutJSONBuilders maps inbound type -> out_json builder key for FillOutJson's
// dispatch. Alias rows are excluded (they are never a real inbound type). An empty
// builder key means "wipe out_json" (FillOutJson's safety-net / not-yet-delivered).
func OutJSONBuilders() map[string]string {
	m := map[string]string{}
	for _, in := range loaded.Inbounds {
		if in.Alias {
			continue
		}
		m[in.Type] = in.OutJSONBuilder
	}
	return m
}

// SkipOutJSONTypes is the set of inbound types FillOutJson returns early for
// (transparent/local inbounds with no client outbound: direct/tun/redirect/tproxy).
func SkipOutJSONTypes() map[string]struct{} {
	m := map[string]struct{}{}
	for _, in := range loaded.Inbounds {
		if in.SkipOutJSON {
			m[in.Type] = struct{}{}
		}
	}
	return m
}

// CredentialMap returns the per-user credential field mapping for an inbound type
// (client.config field -> outbound field), or nil if none. Used by the subscription
// builders to map e.g. name->username / name->user.
func CredentialMap(inboundType string) map[string]string {
	for _, in := range loaded.Inbounds {
		if in.Type == inboundType {
			if in.CredentialMap == nil {
				return nil
			}
			out := make(map[string]string, len(in.CredentialMap))
			for k, v := range in.CredentialMap {
				out[k] = v
			}
			return out
		}
	}
	return nil
}
