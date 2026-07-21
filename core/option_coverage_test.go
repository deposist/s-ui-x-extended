package core

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// optionCoverageTest extracts core option struct fields via AST, compares them
// against panel TypeScript interfaces, and fails on any field that is missing
// from both the TS type and the intentionally-hidden allowlist.
//
// This is the drift-detection anchor for the full-protocol-option-coverage
// feature. When a new option field is added to the fork, this test fails until
// the panel TS type is updated (or the field is added to the allowlist with a
// documented reason).

// --- Go AST extraction (duplicated from scripts/gen-option-coverage.go to
// keep the test self-contained) ---

type covGoField struct {
	JSON     string
	GoType   string
	Embedded bool
}

type covGoStruct struct {
	Fields []covGoField
}

func covExprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return covExprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + covExprString(e.X)
	case *ast.ArrayType:
		return "[]" + covExprString(e.Elt)
	case *ast.MapType:
		return "map[" + covExprString(e.Key) + "]" + covExprString(e.Value)
	case *ast.IndexExpr:
		return covExprString(e.X) + "[" + covExprString(e.Index) + "]"
	case *ast.IndexListExpr:
		parts := make([]string, 0, len(e.Indices))
		for _, idx := range e.Indices {
			parts = append(parts, covExprString(idx))
		}
		return covExprString(e.X) + "[" + strings.Join(parts, ",") + "]"
	default:
		return ""
	}
}

func covJSONName(name string, tagLit *ast.BasicLit) (string, bool) {
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

func covBaseType(t string) string {
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

func covParseStructs(dir string) (map[string]covGoStruct, error) {
	fset := token.NewFileSet()
	structs := map[string]covGoStruct{}
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
				info := covGoStruct{}
				for _, f := range st.Fields.List {
					typeName := covExprString(f.Type)
					if len(f.Names) == 0 {
						info.Fields = append(info.Fields, covGoField{GoType: typeName, Embedded: true})
						continue
					}
					for _, n := range f.Names {
						j, ok := covJSONName(n.Name, f.Tag)
						if !ok {
							continue
						}
						info.Fields = append(info.Fields, covGoField{JSON: j, GoType: typeName})
					}
				}
				structs[ts.Name.Name] = info
			}
		}
		return nil
	})
	return structs, err
}

func covFlatten(name string, structs map[string]covGoStruct, seen map[string]bool) []covGoField {
	if seen[name] {
		return nil
	}
	seen[name] = true
	info, ok := structs[name]
	if !ok {
		return nil
	}
	var out []covGoField
	for _, f := range info.Fields {
		if f.Embedded || f.JSON == "" {
			out = append(out, covFlatten(covBaseType(f.GoType), structs, seen)...)
			continue
		}
		out = append(out, f)
	}
	return out
}

func covFlatFields(name string, structs map[string]covGoStruct) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range covFlatten(name, structs, map[string]bool{}) {
		if f.JSON == "" || seen[f.JSON] {
			continue
		}
		seen[f.JSON] = true
		out = append(out, f.JSON)
	}
	sort.Strings(out)
	return out
}

// --- TS interface extraction (simplified) ---

type covTSInterface struct {
	Extends []string
	Fields  map[string]bool
}

func covParseTSText(text string, result map[string]covTSInterface) {
	for i := 0; i < len(text); {
		idx := strings.Index(text[i:], "interface ")
		if idx < 0 {
			break
		}
		start := i + idx
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
		fields := map[string]bool{}
		for _, line := range strings.Split(body, "\n") {
			s := strings.TrimSpace(line)
			if s == "" || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") {
				continue
			}
			s = strings.TrimPrefix(s, "readonly ")
			for k := 0; k < len(s); k++ {
				c := s[k]
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (c >= '0' && c <= '9' && k > 0)) {
					name := s[:k]
					if name == "" {
						break
					}
					rest := strings.TrimLeft(s[k:], "? \t")
					if strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "(") {
						fields[name] = true
					}
					break
				}
			}
		}
		result[name] = covTSInterface{Extends: extends, Fields: fields}
		i = j
	}
}

func covFlatTS(name string, interfaces map[string]covTSInterface, seen map[string]bool) map[string]bool {
	if seen[name] {
		return map[string]bool{}
	}
	seen[name] = true
	it, ok := interfaces[name]
	if !ok {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for f := range it.Fields {
		out[f] = true
	}
	for _, e := range it.Extends {
		for k := range covFlatTS(e, interfaces, seen) {
			out[k] = true
		}
	}
	return out
}

// --- Allowlist ---

type covAllowlist struct {
	Hidden map[string]map[string]string `json:"hidden"`
}

func covLoadAllowlist(path string) (*covAllowlist, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &covAllowlist{Hidden: map[string]map[string]string{}}, nil
		}
		return nil, err
	}
	var a covAllowlist
	if err := json.Unmarshal(src, &a); err != nil {
		return nil, err
	}
	if a.Hidden == nil {
		a.Hidden = map[string]map[string]string{}
	}
	return &a, nil
}

// --- Mapping ---

type covMapping struct {
	GoStruct    string
	TSInterface string
	Context     string // "in", "out", "ep", "prov"
}

func covMappings() []covMapping {
	return []covMapping{
		// Inbounds
		{"DirectInboundOptions", "Direct", "in"}, {"SocksInboundOptions", "SOCKS", "in"},
		{"HTTPMixedInboundOptions", "Mixed", "in"}, {"ShadowsocksInboundOptions", "Shadowsocks", "in"},
		{"VMessInboundOptions", "VMess", "in"}, {"VLESSInboundOptions", "VLESS", "in"},
		{"TrojanInboundOptions", "Trojan", "in"}, {"NaiveInboundOptions", "Naive", "in"},
		{"HysteriaInboundOptions", "Hysteria", "in"}, {"Hysteria2InboundOptions", "Hysteria2", "in"},
		{"TUICInboundOptions", "TUIC", "in"}, {"AnyTLSInboundOptions", "AnyTls", "in"},
		{"ShadowTLSInboundOptions", "ShadowTLS", "in"}, {"MieruInboundOptions", "Mieru", "in"},
		{"SudokuInboundOptions", "Sudoku", "in"}, {"TrustTunnelInboundOptions", "TrustTunnel", "in"},
		{"SSHInboundOptions", "SSH", "in"}, {"MTProxyInboundOptions", "MTProxy", "in"},
		{"TunInboundOptions", "Tun", "in"}, {"RedirectInboundOptions", "Redirect", "in"},
		{"TProxyInboundOptions", "TProxy", "in"}, {"BondInboundOptions", "BondInbound", "in"},
		{"FailoverInboundOptions", "CoreFailoverInbound", "in"},
		// Outbounds
		{"_DirectOutboundOptions", "Direct", "out"}, {"SOCKSOutboundOptions", "SOCKS", "out"},
		{"HTTPOutboundOptions", "HTTP", "out"}, {"ShadowsocksOutboundOptions", "Shadowsocks", "out"},
		{"VMessOutboundOptions", "VMESS", "out"}, {"VLESSOutboundOptions", "VLESS", "out"},
		{"TrojanOutboundOptions", "Trojan", "out"}, {"NaiveOutboundOptions", "Naive", "out"},
		{"HysteriaOutboundOptions", "Hysteria", "out"}, {"Hysteria2OutboundOptions", "Hysteria2", "out"},
		{"TUICOutboundOptions", "TUIC", "out"}, {"AnyTLSOutboundOptions", "AnyTls", "out"},
		{"ShadowTLSOutboundOptions", "ShadowTLS", "out"}, {"MieruOutboundOptions", "Mieru", "out"},
		{"SudokuOutboundOptions", "Sudoku", "out"}, {"TrustTunnelOutboundOptions", "TrustTunnel", "out"},
		{"SSHOutboundOptions", "SSH", "out"}, {"TorOutboundOptions", "Tor", "out"},
		{"MASQUEOutboundOptions", "MASQUE", "out"}, {"OpenVPNOutboundOptions", "OpenVPN", "out"},
		{"ParserOutboundOptions", "Parser", "out"}, {"SelectorOutboundOptions", "Selector", "out"},
		{"URLTestOutboundOptions", "URLTest", "out"}, {"FallbackOutboundOptions", "Fallback", "out"},
		{"BondOutboundOptions", "Bond", "out"}, {"BandwidthLimiterOutboundOptions", "BandwidthLimiter", "out"},
		{"ConnectionLimiterOutboundOptions", "ConnectionLimiter", "out"},
		{"TrafficLimiterOutboundOptions", "TrafficLimiter", "out"},
		{"RateLimiterOutboundOptions", "RateLimiter", "out"},
		{"FailoverOutboundOptions", "CoreFailover", "out"}, {"StubOptions", "Block", "out"},
		// Endpoints
		{"WireGuardEndpointOptions", "WireGuard", "ep"}, {"WARPEndpointOptions", "Warp", "ep"},
		{"TailscaleEndpointOptions", "Tailscale", "ep"}, {"VPNServerEndpointOptions", "VpnServer", "ep"},
		{"VPNClientEndpointOptions", "VpnClient", "ep"},
		// Providers
		{"ProviderInlineOptions", "ProviderInline", "prov"},
		{"ProviderLocalOptions", "ProviderLocal", "prov"},
		{"ProviderRemoteOptions", "ProviderRemote", "prov"},
	}
}

var covPanelEntityFields = map[string]bool{
	"tag": true, "type": true, "id": true,
}

func covMergeTS(base map[string]covTSInterface, file string) map[string]covTSInterface {
	out := make(map[string]covTSInterface, len(base))
	for k, v := range base {
		out[k] = v
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return out // file may not exist in some contexts
	}
	covParseTSText(string(src), out)
	return out
}

// --- Test ---

func TestOptionCoverageNoMissingFields(t *testing.T) {
	// Find the fork option directory at test time.
	cmd := exec.Command("go", "list", "-f", "{{.Dir}}", "github.com/sagernet/sing-box/option")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("cannot resolve option dir: %v", err)
	}
	optionDir := strings.TrimSpace(string(out))
	if optionDir == "" {
		t.Skip("option dir is empty")
	}

	// Parse Go option structs.
	goStructs, err := covParseStructs(optionDir)
	if err != nil {
		t.Fatalf("parse go structs: %v", err)
	}

	// Parse TS interfaces with context separation to avoid name collisions
	// (e.g. inbound Sudoku extends InboundBasics, outbound Sudoku extends OutboundBasics).
	sharedTS := map[string]covTSInterface{}
	for _, f := range []string{"../frontend/src/types/dial.ts", "../frontend/src/types/tls.ts", "../frontend/src/types/multiplex.ts", "../frontend/src/types/transport.ts"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		covParseTSText(string(src), sharedTS)
	}
	inboundTS := covMergeTS(sharedTS, "../frontend/src/types/inbounds.ts")
	outboundTS := covMergeTS(sharedTS, "../frontend/src/types/outbounds.ts")
	endpointTS := covMergeTS(sharedTS, "../frontend/src/types/endpoints.ts")
	providerTS := covMergeTS(covMergeTS(sharedTS, "../frontend/src/types/outbounds.ts"), "../frontend/src/types/providers.ts")

	// Load allowlist.
	allowlist, err := covLoadAllowlist("capabilities/intentionally-hidden.json")
	if err != nil {
		t.Fatalf("load allowlist: %v", err)
	}

	// Check each mapping.
	var missing []string
	for _, m := range covMappings() {
		var tsInterfaces map[string]covTSInterface
		switch m.Context {
		case "in":
			tsInterfaces = inboundTS
		case "out":
			tsInterfaces = outboundTS
		case "ep":
			tsInterfaces = endpointTS
		case "prov":
			tsInterfaces = providerTS
		}
		goFields := covFlatFields(m.GoStruct, goStructs)
		tsFields := covFlatTS(m.TSInterface, tsInterfaces, map[string]bool{})
		for _, field := range goFields {
			if covPanelEntityFields[field] {
				continue
			}
			if tsFields[field] {
				continue
			}
			if _, ok := allowlist.Hidden[m.GoStruct][field]; ok {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s.%s", m.GoStruct, field))
		}
	}

	if len(missing) > 0 {
		t.Errorf("option coverage drift: %d field(s) missing from panel TS types (not in allowlist):\n%s",
			len(missing), strings.Join(missing, "\n"))
		t.Logf("to fix: add these fields to the corresponding TS interface in frontend/src/types/*.ts, or add them to core/capabilities/intentionally-hidden.json with a reason")
	}
}
