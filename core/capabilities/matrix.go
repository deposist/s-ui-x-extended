package capabilities

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMatrix renders docs/protocol-matrix.md from the manifest: one row per
// protocol type across all categories (inbound/outbound/endpoint/service) with the
// inbound delivery facts and build-tag / gap notes. Kept in sync by a golden test.
func RenderMatrix() string {
	type row struct {
		typ                             string
		in, out, endpoint, service      bool
		tlsTmpl, users                  bool
		clientDelivery, buildTag, notes string
		order                           int
	}
	rows := map[string]*row{}
	order := 0
	get := func(typ string) *row {
		r, ok := rows[typ]
		if !ok {
			r = &row{typ: typ, order: order}
			order++
			rows[typ] = r
		}
		return r
	}
	mergeNote := func(r *row, n string) {
		if n == "" {
			return
		}
		if r.notes == "" {
			r.notes = n
		} else if !strings.Contains(r.notes, n) {
			r.notes += "; " + n
		}
	}

	for _, in := range loaded.Inbounds {
		if in.Alias {
			continue
		}
		r := get(in.Type)
		r.in = true
		r.tlsTmpl = in.HasTLSTemplate
		r.users = in.HasUsers
		r.clientDelivery = in.ClientDelivery
		if in.BuildTag != "" {
			r.buildTag = in.BuildTag
		}
		mergeNote(r, in.Notes)
	}
	for _, o := range loaded.Outbounds {
		r := get(o.Type)
		r.out = true
		if r.buildTag == "" {
			r.buildTag = o.BuildTag
		}
		mergeNote(r, o.Notes)
	}
	for _, e := range loaded.Endpoints {
		r := get(e.Type)
		r.endpoint = true
		if r.buildTag == "" {
			r.buildTag = e.BuildTag
		}
		mergeNote(r, e.Notes)
	}
	for _, s := range loaded.Services {
		r := get(s.Type)
		r.service = true
		if r.buildTag == "" {
			r.buildTag = s.BuildTag
		}
		mergeNote(r, s.Notes)
	}

	ordered := make([]*row, 0, len(rows))
	for _, r := range rows {
		ordered = append(ordered, r)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].order < ordered[j].order })

	mark := func(b bool) string {
		if b {
			return "✓"
		}
		return "–"
	}

	var sb strings.Builder
	sb.WriteString("<!-- GENERATED from core/capabilities/protocols.json by RenderMatrix (capabilities/matrix.go).\n")
	sb.WriteString("     Do not edit by hand; regenerate with: go test ./core/capabilities -run TestProtocolMatrixDoc -update -->\n\n")
	sb.WriteString("# Protocol capability matrix\n\n")
	sb.WriteString("Single source of truth: `core/capabilities/protocols.json`. Derived backend maps, ")
	sb.WriteString("frontend lists, and this matrix all come from it.\n\n")
	sb.WriteString("| Type | in | out | endpoint | service | tls-tmpl | users | clientDelivery | buildTag | notes/gap |\n")
	sb.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range ordered {
		delivery := r.clientDelivery
		if delivery == "" {
			delivery = "–"
		}
		bt := r.buildTag
		if bt == "" {
			bt = "–"
		}
		notes := r.notes
		if notes == "" {
			notes = "—"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			r.typ, mark(r.in), mark(r.out), mark(r.endpoint), mark(r.service),
			mark(r.tlsTmpl), mark(r.users), delivery, bt, notes))
	}
	return sb.String()
}
