package capabilities

import "sort"

// compiledBuildTags is populated by the per-tag buildtag_with_*.go files: each is
// constrained to `//go:build with_X` and flips its entry true at init when that tag
// is compiled in. A tag whose file is not compiled simply stays absent (false).
// Detection is therefore by what the compiler actually included — never by parsing
// build.sh — so it cannot lie about the running binary.
var compiledBuildTags = map[string]bool{}

// knownBuildTags is the canonical set reported by BuildTags(), so absent tags are
// reported explicitly as false rather than missing. Keep in sync with the
// buildtag_with_*.go files.
var knownBuildTags = []string{
	"with_quic", "with_grpc", "with_utls", "with_acme", "with_gvisor",
	"with_tailscale", "with_dhcp", "with_wireguard", "with_masque", "with_mtproxy",
	"with_openvpn", "with_sudoku", "with_trusttunnel", "with_ccm", "with_ocm",
	"with_oomkiller", "with_naive_outbound", "with_profiler",
}

// BuildTags reports every known build tag with whether it was compiled into this
// binary. Values are pure booleans — no paths, versions or secrets — so the result
// is safe to return from the admin API without aiding fingerprinting beyond the
// feature set the operator already controls.
func BuildTags() map[string]bool {
	out := make(map[string]bool, len(knownBuildTags))
	for _, t := range knownBuildTags {
		out[t] = compiledBuildTags[t]
	}
	return out
}

// tagCompiled reports whether a build tag is compiled in. An empty tag (a protocol
// that needs no tag) is always considered available.
func tagCompiled(tag string) bool {
	if tag == "" {
		return true
	}
	return compiledBuildTags[tag]
}

// APIInbound is the admin-safe per-inbound capability view returned by the
// /api/capabilities endpoint. It carries only UI capability flags and the
// build-availability of the type — no builder names, paths or secrets.
type APIInbound struct {
	Type           string `json:"type"`
	ClientDelivery string `json:"clientDelivery"`
	HasUsers       bool   `json:"hasUsers"`
	HasInData      bool   `json:"hasInData"`
	HasTLSTemplate bool   `json:"hasTlsTemplate"`
	MuxAvailable   bool   `json:"muxAvailable"`
	OnlyTLS        bool   `json:"onlyTls"`
	BuildTag       string `json:"buildTag,omitempty"`
	Available      bool   `json:"available"`
}

// APIView is the response body for the admin-only /api/capabilities endpoint: the
// compiled build-tag flags plus, per inbound/outbound/group type, whether it is
// available in this build. Alias rows (e.g. shadowsocks16) are excluded.
type APIView struct {
	BuildTags map[string]bool `json:"buildTags"`
	Inbounds  []APIInbound    `json:"inbounds"`
	Outbounds []APIOutbound   `json:"outbounds"`
	Groups    []APIGroup      `json:"groups"`
	Providers []APIProvider   `json:"providers"`
}

type APIOutbound struct {
	Type      string `json:"type"`
	BuildTag  string `json:"buildTag"`
	Available bool   `json:"available"`
}

type APIGroup struct {
	Type            string `json:"type"`
	AssembledAs     string `json:"assembledAs,omitempty"`
	PanelManaged    bool   `json:"panelManaged,omitempty"`
	SessionRecovery bool   `json:"sessionRecovery"`
	Available       bool   `json:"available"`
}

type APIProvider struct {
	Type      string `json:"type"`
	BuildTag  string `json:"buildTag,omitempty"`
	Available bool   `json:"available"`
}

// BuildAPIView assembles the admin-safe capability view from the manifest and the
// compiled build tags.
func BuildAPIView() APIView {
	view := APIView{BuildTags: BuildTags()}
	for _, in := range loaded.Inbounds {
		if in.Alias {
			continue
		}
		view.Inbounds = append(view.Inbounds, APIInbound{
			Type:           in.Type,
			ClientDelivery: in.ClientDelivery,
			HasUsers:       in.HasUsers,
			HasInData:      in.HasInData,
			HasTLSTemplate: in.HasTLSTemplate,
			MuxAvailable:   in.MuxAvailable,
			OnlyTLS:        in.OnlyTLS,
			BuildTag:       in.BuildTag,
			Available:      tagCompiled(in.BuildTag),
		})
	}
	for _, o := range loaded.Outbounds {
		view.Outbounds = append(view.Outbounds, APIOutbound{
			Type:      o.Type,
			BuildTag:  o.BuildTag,
			Available: tagCompiled(o.BuildTag),
		})
	}
	for _, g := range loaded.Groups {
		view.Groups = append(view.Groups, APIGroup{
			Type:            g.Type,
			AssembledAs:     g.AssembledAs,
			PanelManaged:    g.PanelManaged,
			SessionRecovery: g.SessionRecovery,
			Available:       true, // group types are always available (assembled from core types)
		})
	}
	for _, p := range loaded.Providers {
		view.Providers = append(view.Providers, APIProvider{
			Type:      p.Type,
			BuildTag:  p.BuildTag,
			Available: tagCompiled(p.BuildTag),
		})
	}
	sort.SliceStable(view.Inbounds, func(i, j int) bool { return view.Inbounds[i].Type < view.Inbounds[j].Type })
	sort.SliceStable(view.Outbounds, func(i, j int) bool { return view.Outbounds[i].Type < view.Outbounds[j].Type })
	sort.SliceStable(view.Groups, func(i, j int) bool { return view.Groups[i].Type < view.Groups[j].Type })
	sort.SliceStable(view.Providers, func(i, j int) bool { return view.Providers[i].Type < view.Providers[j].Type })
	return view
}
