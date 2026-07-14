// Command gen-option-coverage produces a machine-readable and human-readable
// audit of core option struct fields versus panel TypeScript types and Vue UI
// components. It is the drift-detection anchor for the full-protocol-option-
// coverage feature.
//
// Usage:
//
//	go run scripts/gen-option-coverage.go "$(go list -f '{{.Dir}}' github.com/sagernet/sing-box/option)"
//
// Outputs two files:
//   - docs/option-coverage-matrix.json (machine-readable)
//   - docs/option-coverage-matrix.md   (human-readable)
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Go option struct extraction
// ---------------------------------------------------------------------------

type GoField struct {
	JSON     string `json:"json"`
	GoType   string `json:"goType"`
	Embedded bool   `json:"embedded"`
}

type GoStruct struct {
	File   string    `json:"file"`
	Fields []GoField `json:"fields"`
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.ArrayType:
		return "[]" + exprString(e.Elt)
	case *ast.MapType:
		return "map[" + exprString(e.Key) + "]" + exprString(e.Value)
	case *ast.IndexExpr:
		return exprString(e.X) + "[" + exprString(e.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, 0, len(e.Indices))
		for _, idx := range e.Indices {
			parts = append(parts, exprString(idx))
		}
		return exprString(e.X) + "[" + strings.Join(parts, ",") + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func jsonName(name string, tagLit *ast.BasicLit) (string, bool) {
	if tagLit == nil {
		return "", false
	}
	tag := strings.Trim(tagLit.Value, "`")
	j := reflect.StructTag(tag).Get("json")
	if j == "" || j == "-" {
		return "", false
	}
	parts := strings.Split(j, ",")
	if parts[0] == "" {
		return name, true
	}
	return parts[0], true
}

func baseType(t string) string {
	t = strings.TrimPrefix(t, "*")
	for strings.HasPrefix(t, "[]") {
		t = strings.TrimPrefix(t, "[]")
	}
	if strings.Contains(t, "[") {
		t = strings.Split(t, "[")[0]
	}
	if strings.Contains(t, ".") {
		parts := strings.Split(t, ".")
		t = parts[len(parts)-1]
	}
	return t
}

func parseGoStructs(dir string) (map[string]GoStruct, error) {
	fset := token.NewFileSet()
	structs := map[string]GoStruct{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				info := GoStruct{File: filepath.Base(path)}
				for _, f := range st.Fields.List {
					typeName := exprString(f.Type)
					if len(f.Names) == 0 {
						info.Fields = append(info.Fields, GoField{
							JSON:     "",
							GoType:   typeName,
							Embedded: true,
						})
						continue
					}
					for _, n := range f.Names {
						j, ok := jsonName(n.Name, f.Tag)
						if !ok {
							continue
						}
						info.Fields = append(info.Fields, GoField{
							JSON:   j,
							GoType: typeName,
						})
					}
				}
				structs[ts.Name.Name] = info
			}
		}
		return nil
	})
	return structs, err
}

func flattenStruct(name string, structs map[string]GoStruct, seen map[string]bool) []GoField {
	if seen[name] {
		return nil
	}
	seen[name] = true
	info, ok := structs[name]
	if !ok {
		return nil
	}
	var out []GoField
	for _, f := range info.Fields {
		if f.Embedded || f.JSON == "" {
			out = append(out, flattenStruct(baseType(f.GoType), structs, seen)...)
			continue
		}
		out = append(out, f)
	}
	return out
}

func flatFields(name string, structs map[string]GoStruct) []string {
	seenFields := map[string]bool{}
	var out []string
	for _, f := range flattenStruct(name, structs, map[string]bool{}) {
		if f.JSON == "" || seenFields[f.JSON] {
			continue
		}
		seenFields[f.JSON] = true
		out = append(out, f.JSON)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// TypeScript interface extraction
// ---------------------------------------------------------------------------

type TSInterface struct {
	Extends []string `json:"extends"`
	Fields  []string `json:"fields"`
}

func parseTSInterfaces(dir string) (map[string]TSInterface, error) {
	result := map[string]TSInterface{}
	files, err := filepath.Glob(filepath.Join(dir, "*.ts"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(src)
		parseTSFile(text, result)
	}
	return result, nil
}

func parseTSFile(text string, result map[string]TSInterface) {
	for i := 0; i < len(text); {
		idx := strings.Index(text[i:], "interface ")
		if idx < 0 {
			break
		}
		start := i + idx
		// Parse name and optional extends
		rest := text[start+len("interface "):]
		braceIdx := strings.Index(rest, "{")
		if braceIdx < 0 {
			break
		}
		header := rest[:braceIdx]
		nameEnd := strings.IndexAny(header, " \t")
		if nameEnd < 0 {
			i = start + len("interface ") + braceIdx + 1
			continue
		}
		name := strings.TrimSpace(header[:nameEnd])
		extends := []string{}
		extIdx := strings.Index(header, "extends ")
		if extIdx >= 0 {
			extPart := header[extIdx+len("extends "):]
			for _, e := range strings.Split(extPart, ",") {
				e = strings.TrimSpace(e)
				e = strings.Split(e, "<")[0]
				if e != "" {
					extends = append(extends, e)
				}
			}
		}
		// Find matching close brace
		bodyStart := start + len("interface ") + braceIdx + 1
		depth := 1
		j := bodyStart
		for j < len(text) && depth > 0 {
			if text[j] == '{' {
				depth++
			} else if text[j] == '}' {
				depth--
			}
			j++
		}
		body := text[bodyStart : j-1]
		fields := extractTSFields(body)
		result[name] = TSInterface{Extends: extends, Fields: fields}
		i = j
	}
}

func extractTSFields(body string) []string {
	var fields []string
	depth := 0
	var line strings.Builder
	for _, ch := range body {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
		if ch == '\n' && depth == 0 {
			s := strings.TrimSpace(line.String())
			line.Reset()
			if s == "" || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") {
				continue
			}
			// Match: fieldName? : type  OR  fieldName? : {
			// Also match: fieldName?(...): type (methods)
			fieldName := extractTSFieldName(s)
			if fieldName != "" {
				fields = append(fields, fieldName)
			}
		} else {
			line.WriteRune(ch)
		}
	}
	s := strings.TrimSpace(line.String())
	if s != "" {
		fieldName := extractTSFieldName(s)
		if fieldName != "" {
			fields = append(fields, fieldName)
		}
	}
	return fields
}

func extractTSFieldName(s string) string {
	// Remove leading modifiers
	s = strings.TrimPrefix(s, "readonly ")
	// Find the field name: first identifier optionally followed by ?
	// Pattern: name? :  or  name?:  or  name :
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (c >= '0' && c <= '9' && i > 0)) {
			// Read the identifier
			name := s[:i]
			if name == "" {
				return ""
			}
			// Skip ? if present
			rest := strings.TrimLeft(s[i:], "? \t")
			if strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "(") {
				return name
			}
			return ""
		}
	}
	// Entire string is an identifier (unlikely for a field)
	return ""
}

func flatTSFields(name string, interfaces map[string]TSInterface, seen map[string]bool) map[string]bool {
	if seen[name] {
		return map[string]bool{}
	}
	seen[name] = true
	it, ok := interfaces[name]
	if !ok {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, f := range it.Fields {
		out[f] = true
	}
	for _, e := range it.Extends {
		for k := range flatTSFields(e, interfaces, seen) {
			out[k] = true
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Vue component v-model extraction
// ---------------------------------------------------------------------------

func parseVueModels(dir string) (map[string][]string, error) {
	result := map[string][]string{}
	files, err := filepath.Glob(filepath.Join(dir, "*.vue"))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(src)
		base := filepath.Base(path)
		var models []string
		// Match v-model="..." patterns
		for _, m := range allMatches(`v-model="([^"]+)"`, text) {
			field := extractVModelField(m)
			if field != "" {
				models = append(models, field)
			}
		}
		// Also match v-model:prop="..." patterns
		for _, m := range allMatches(`v-model:[\w-]+="([^"]+)"`, text) {
			field := extractVModelField(m)
			if field != "" {
				models = append(models, field)
			}
		}
		// Match :value="..." or :model-value="..." as fallback
		for _, m := range allMatches(`:(?:model-)?value="([^"]+)"`, text) {
			field := extractVModelField(m)
			if field != "" {
				models = append(models, field)
			}
		}
		result[base] = uniqueStrings(models)
	}
	return result, nil
}

func allMatches(pattern, text string) []string {
	return regexpFindAll(pattern, text)
}

func regexpFindAll(pattern, text string) []string {
	// Simple v-model extraction without regexp package
	// We look for v-model="..." and extract the content
	var out []string
	search := "v-model=\""
	if strings.Contains(pattern, "v-model") {
		for i := 0; i < len(text); {
			idx := strings.Index(text[i:], search)
			if idx < 0 {
				break
			}
			start := i + idx + len(search)
			end := strings.Index(text[start:], "\"")
			if end < 0 {
				break
			}
			out = append(out, text[start:start+end])
			i = start + end + 1
		}
	}
	// v-model:prop="..."
	search2 := "v-model:"
	for i := 0; i < len(text); {
		idx := strings.Index(text[i:], search2)
		if idx < 0 {
			break
		}
		start := i + idx + len(search2)
		// Skip prop name until ="
		eqIdx := strings.Index(text[start:], "=\"")
		if eqIdx < 0 {
			break
		}
		valStart := start + eqIdx + 2
		end := strings.Index(text[valStart:], "\"")
		if end < 0 {
			break
		}
		out = append(out, text[valStart:valStart+end])
		i = valStart + end + 1
	}
	return out
}

func extractVModelField(expr string) string {
	// v-model="data.field_name" → field_name
	// v-model="someComputed" → someComputed
	// v-model="$props.outbound.server" → server
	parts := strings.Split(expr, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Struct → TS interface mapping
// ---------------------------------------------------------------------------

type StructMapping struct {
	GoStruct    string
	TSInterface string
	Context     string // "in", "out", "ep", "prov"
}

func buildStructMappings() []StructMapping {
	var m []StructMapping
	// Inbounds
	for gs, ts := range map[string]string{
		"DirectInboundOptions":      "Direct",
		"SocksInboundOptions":       "SOCKS",
		"HTTPMixedInboundOptions":   "Mixed",
		"ShadowsocksInboundOptions": "Shadowsocks",
		"VMessInboundOptions":       "VMess",
		"VLESSInboundOptions":       "VLESS",
		"TrojanInboundOptions":      "Trojan",
		"NaiveInboundOptions":       "Naive",
		"HysteriaInboundOptions":    "Hysteria",
		"Hysteria2InboundOptions":   "Hysteria2",
		"TUICInboundOptions":        "TUIC",
		"AnyTLSInboundOptions":      "AnyTls",
		"ShadowTLSInboundOptions":   "ShadowTLS",
		"MieruInboundOptions":       "Mieru",
		"SudokuInboundOptions":      "Sudoku",
		"TrustTunnelInboundOptions": "TrustTunnel",
		"SSHInboundOptions":         "SSH",
		"MTProxyInboundOptions":     "MTProxy",
		"TunInboundOptions":         "Tun",
		"RedirectInboundOptions":    "Redirect",
		"TProxyInboundOptions":      "TProxy",
		"BondInboundOptions":        "BondInbound",
		"FailoverInboundOptions":    "CoreFailoverInbound",
	} {
		m = append(m, StructMapping{gs, ts, "in"})
	}
	// Outbounds
	for gs, ts := range map[string]string{
		"_DirectOutboundOptions":           "Direct",
		"SOCKSOutboundOptions":             "SOCKS",
		"HTTPOutboundOptions":              "HTTP",
		"ShadowsocksOutboundOptions":       "Shadowsocks",
		"VMessOutboundOptions":             "VMESS",
		"VLESSOutboundOptions":             "VLESS",
		"TrojanOutboundOptions":            "Trojan",
		"NaiveOutboundOptions":             "Naive",
		"HysteriaOutboundOptions":          "Hysteria",
		"Hysteria2OutboundOptions":         "Hysteria2",
		"TUICOutboundOptions":              "TUIC",
		"AnyTLSOutboundOptions":            "AnyTls",
		"ShadowTLSOutboundOptions":         "ShadowTLS",
		"MieruOutboundOptions":             "Mieru",
		"SudokuOutboundOptions":            "Sudoku",
		"TrustTunnelOutboundOptions":       "TrustTunnel",
		"SSHOutboundOptions":               "SSH",
		"TorOutboundOptions":               "Tor",
		"MASQUEOutboundOptions":            "MASQUE",
		"OpenVPNOutboundOptions":           "OpenVPN",
		"ParserOutboundOptions":            "Parser",
		"SelectorOutboundOptions":          "Selector",
		"URLTestOutboundOptions":           "URLTest",
		"FallbackOutboundOptions":          "Fallback",
		"BondOutboundOptions":              "Bond",
		"BandwidthLimiterOutboundOptions":  "BandwidthLimiter",
		"ConnectionLimiterOutboundOptions": "ConnectionLimiter",
		"TrafficLimiterOutboundOptions":    "TrafficLimiter",
		"RateLimiterOutboundOptions":       "RateLimiter",
		"FailoverOutboundOptions":          "CoreFailover",
		"StubOptions":                      "Block",
	} {
		m = append(m, StructMapping{gs, ts, "out"})
	}
	// Endpoints
	for gs, ts := range map[string]string{
		"WireGuardEndpointOptions": "WireGuard",
		"WARPEndpointOptions":      "Warp",
		"TailscaleEndpointOptions": "Tailscale",
		"VPNServerEndpointOptions": "VpnServer",
		"VPNClientEndpointOptions": "VpnClient",
	} {
		m = append(m, StructMapping{gs, ts, "ep"})
	}
	// Providers
	for gs, ts := range map[string]string{
		"ProviderInlineOptions": "ProviderInline",
		"ProviderLocalOptions":  "ProviderLocal",
		"ProviderRemoteOptions": "ProviderRemote",
	} {
		m = append(m, StructMapping{gs, ts, "prov"})
	}
	return m
}

// ---------------------------------------------------------------------------
// Coverage matrix generation
// ---------------------------------------------------------------------------

type MatrixEntry struct {
	Struct      string `json:"struct"`
	Field       string `json:"field"`
	TSInterface string `json:"tsInterface,omitempty"`
	UIComponent string `json:"uiComponent,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
}

type Allowlist struct {
	Hidden map[string]map[string]string `json:"hidden"`
}

func loadAllowlist(path string) (*Allowlist, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Allowlist{Hidden: map[string]map[string]string{}}, nil
		}
		return nil, err
	}
	var a Allowlist
	if err := json.Unmarshal(src, &a); err != nil {
		return nil, err
	}
	if a.Hidden == nil {
		a.Hidden = map[string]map[string]string{}
	}
	return &a, nil
}

func tsContextDir(context string) string {
	switch context {
	case "in":
		return "frontend/src/types/inbounds.ts"
	case "out":
		return "frontend/src/types/outbounds.ts"
	case "ep":
		return "frontend/src/types/endpoints.ts"
	case "prov":
		return "frontend/src/types/providers.ts"
	default:
		return ""
	}
}

func parseTSFileInto(path string, result map[string]TSInterface) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	parseTSFile(string(src), result)
	return nil
}

func cloneTSInterfaces(in map[string]TSInterface) map[string]TSInterface {
	out := make(map[string]TSInterface, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mustMergeTS(base map[string]TSInterface, paths ...string) (map[string]TSInterface, error) {
	out := cloneTSInterfaces(base)
	for _, path := range paths {
		if err := parseTSFileInto(path, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func vueComponentForTS(tsName string) string {
	// Try to find a Vue component that matches the TS interface name
	// by checking files in frontend/src/components/protocols/
	componentsDir := "frontend/src/components/protocols"
	files, err := os.ReadDir(componentsDir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".vue") {
			continue
		}
		base := strings.TrimSuffix(f.Name(), ".vue")
		// Match by uiEditor convention from manifest, or by name similarity
		if base == tsName || base == tsName+"Inbound" {
			return f.Name()
		}
	}
	return ""
}

var panelEntityFields = map[string]bool{
	"tag": true, "type": true, "id": true,
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-option-coverage <option-dir>")
		os.Exit(1)
	}
	optionDir := os.Args[1]

	// 1. Parse Go option structs
	goStructs, err := parseGoStructs(optionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse go structs: %v\n", err)
		os.Exit(1)
	}

	// 2. Parse TS interfaces with context separation. Inbound and outbound files
	// intentionally reuse interface names (for example Sudoku and AnyTls), so a
	// single global map would overwrite one context with another and produce false
	// missing rows.
	sharedTS := map[string]TSInterface{}
	for _, path := range []string{
		"frontend/src/types/dial.ts",
		"frontend/src/types/tls.ts",
		"frontend/src/types/multiplex.ts",
		"frontend/src/types/transport.ts",
	} {
		if err := parseTSFileInto(path, sharedTS); err != nil {
			fmt.Fprintf(os.Stderr, "parse shared ts %s: %v\n", path, err)
			os.Exit(1)
		}
	}
	inboundTS, err := mustMergeTS(sharedTS, "frontend/src/types/inbounds.ts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse inbound ts: %v\n", err)
		os.Exit(1)
	}
	outboundTS, err := mustMergeTS(sharedTS, "frontend/src/types/outbounds.ts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse outbound ts: %v\n", err)
		os.Exit(1)
	}
	endpointTS, err := mustMergeTS(sharedTS, "frontend/src/types/endpoints.ts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse endpoint ts: %v\n", err)
		os.Exit(1)
	}
	providerTS, err := mustMergeTS(sharedTS, "frontend/src/types/outbounds.ts", "frontend/src/types/providers.ts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse provider ts: %v\n", err)
		os.Exit(1)
	}
	tsByContext := map[string]map[string]TSInterface{
		"in":   inboundTS,
		"out":  outboundTS,
		"ep":   endpointTS,
		"prov": providerTS,
	}

	// 3. Parse Vue components
	vueModels, err := parseVueModels("frontend/src/components/protocols")
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse vue models: %v\n", err)
		os.Exit(1)
	}

	// 4. Load allowlist
	allowlist, err := loadAllowlist("core/capabilities/intentionally-hidden.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load allowlist: %v\n", err)
		os.Exit(1)
	}

	// 5. Build coverage matrix
	mappings := buildStructMappings()
	var entries []MatrixEntry
	for _, m := range mappings {
		goFields := flatFields(m.GoStruct, goStructs)
		tsFields := flatTSFields(m.TSInterface, tsByContext[m.Context], map[string]bool{})
		uiComp := vueComponentForTS(m.TSInterface)
		uiFieldSet := map[string]bool{}
		if uiComp != "" {
			for _, f := range vueModels[uiComp] {
				uiFieldSet[f] = true
			}
		}
		for _, field := range goFields {
			if panelEntityFields[field] {
				continue
			}
			entry := MatrixEntry{
				Struct: m.GoStruct,
				Field:  field,
			}
			if tsFields[field] {
				entry.TSInterface = m.TSInterface
			}
			if uiFieldSet[field] {
				entry.UIComponent = uiComp
			}
			// Determine status
			if reason, ok := allowlist.Hidden[m.GoStruct][field]; ok {
				entry.Status = "intentionally-hidden"
				entry.Reason = reason
			} else if tsFields[field] && uiFieldSet[field] {
				entry.Status = "covered"
			} else if tsFields[field] {
				entry.Status = "typed-only"
			} else {
				entry.Status = "missing"
			}
			entries = append(entries, entry)
		}
	}

	// 6. Output JSON
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal json: %v\n", err)
		os.Exit(1)
	}
	// #nosec G306 -- generated documentation is intentionally world-readable.
	if err := os.WriteFile("docs/option-coverage-matrix.json", jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write json: %v\n", err)
		os.Exit(1)
	}

	// 7. Output Markdown
	var md strings.Builder
	md.WriteString("# Option Coverage Matrix\n\n")
	md.WriteString("Generated by `scripts/gen-option-coverage.go`. Do not edit by hand.\n\n")

	// Group by struct
	byStruct := map[string][]MatrixEntry{}
	for _, e := range entries {
		byStruct[e.Struct] = append(byStruct[e.Struct], e)
	}
	structNames := make([]string, 0, len(byStruct))
	for k := range byStruct {
		structNames = append(structNames, k)
	}
	sort.Strings(structNames)

	missingCount := 0
	coveredCount := 0
	hiddenCount := 0
	typedOnlyCount := 0

	for _, s := range structNames {
		fieldEntries := byStruct[s]
		md.WriteString(fmt.Sprintf("## %s\n\n", s))
		md.WriteString("| Field | TS type | UI component | Status | Reason |\n")
		md.WriteString("|-------|---------|--------------|--------|--------|\n")
		for _, e := range fieldEntries {
			md.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				e.Field, e.TSInterface, e.UIComponent, e.Status, e.Reason))
			switch e.Status {
			case "missing":
				missingCount++
			case "covered":
				coveredCount++
			case "intentionally-hidden":
				hiddenCount++
			case "typed-only":
				typedOnlyCount++
			}
		}
		md.WriteString("\n")
	}

	md.WriteString("## Summary\n\n")
	md.WriteString("| Status | Count |\n|--------|-------|\n")
	md.WriteString(fmt.Sprintf("| covered | %d |\n", coveredCount))
	md.WriteString(fmt.Sprintf("| typed-only | %d |\n", typedOnlyCount))
	md.WriteString(fmt.Sprintf("| intentionally-hidden | %d |\n", hiddenCount))
	md.WriteString(fmt.Sprintf("| missing | %d |\n", missingCount))

	// #nosec G306 -- generated documentation is intentionally world-readable.
	if err := os.WriteFile("docs/option-coverage-matrix.md", []byte(md.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write md: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("coverage matrix: %d entries, %d covered, %d typed-only, %d hidden, %d missing\n",
		len(entries), coveredCount, typedOnlyCount, hiddenCount, missingCount)
	fmt.Println("output: docs/option-coverage-matrix.json, docs/option-coverage-matrix.md")
}
