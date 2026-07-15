package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func amneziaOptions(t *testing.T, amnezia map[string]any) json.RawMessage {
	t.Helper()
	options, err := json.Marshal(map[string]any{"amnezia": amnezia})
	if err != nil {
		t.Fatal(err)
	}
	return options
}

func TestParseAWGHeaderValue(t *testing.T) {
	cases := []struct {
		name    string
		value   any
		want    *awgHeaderRange
		wantErr bool
	}{
		{"absent", nil, nil, false},
		{"empty string", "  ", nil, false},
		{"single number", float64(1000), &awgHeaderRange{1000, 1000}, false},
		{"single int", int(77), &awgHeaderRange{77, 77}, false},
		{"single string", "12345", &awgHeaderRange{12345, 12345}, false},
		{"range string", "1000-2000", &awgHeaderRange{1000, 2000}, false},
		{"max uint32", float64(4294967295), &awgHeaderRange{4294967295, 4294967295}, false},
		{"reversed range", "2000-1000", nil, true},
		{"negative number", float64(-5), nil, true},
		{"fractional number", 10.5, nil, true},
		{"over uint32", "4294967296", nil, true},
		{"three parts", "1-2-3", nil, true},
		{"garbage", "abc", nil, true},
		{"spaces in range", "10 - 20", nil, true},
		{"bool value", true, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAWGHeaderValue("H1", tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseAWGHeaderValue(%v) expected error, got %v", tc.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAWGHeaderValue(%v) unexpected error: %v", tc.value, err)
			}
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("parseAWGHeaderValue(%v) = %v, want %v", tc.value, got, tc.want)
			}
			if got != nil && *got != *tc.want {
				t.Fatalf("parseAWGHeaderValue(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestValidateAmneziaOptionsTable(t *testing.T) {
	valid := map[string]any{
		"jc": 4, "jmin": 40, "jmax": 90,
		"s1": 15, "s2": 20, "s3": 12, "s4": 8,
		"h1": "1000-1099", "h2": float64(2000), "h3": "3000-3099", "h4": "4000-4099",
	}
	cases := []struct {
		name     string
		mutate   func(map[string]any)
		errChunk string
	}{
		{"valid set", func(m map[string]any) {}, ""},
		{"legacy default h1-h4 reserved", func(m map[string]any) {
			m["h1"], m["h2"], m["h3"], m["h4"] = float64(1), float64(2), float64(3), float64(4)
		}, "reserved"},
		{"h zero reserved", func(m map[string]any) { m["h1"] = float64(0) }, "reserved"},
		{"h range touching reserved", func(m map[string]any) { m["h1"] = "4-100" }, "reserved"},
		{"h range from 5 ok", func(m map[string]any) { m["h1"] = "5-100" }, ""},
		{"identical single headers", func(m map[string]any) {
			m["h1"], m["h2"] = float64(9999), float64(9999)
		}, "overlap"},
		{"overlapping ranges", func(m map[string]any) {
			m["h1"], m["h2"] = "1000-2000", "1500-2500"
		}, "overlap"},
		{"touching range edges overlap", func(m map[string]any) {
			m["h1"], m["h2"] = "1000-2000", "2000-3000"
		}, "overlap"},
		{"single inside range overlaps", func(m map[string]any) {
			m["h1"], m["h2"] = "1000-2000", float64(1500)
		}, "overlap"},
		{"adjacent ranges ok", func(m map[string]any) {
			m["h1"], m["h2"] = "1000-2000", "2001-2999"
		}, ""},
		{"partial headers ok", func(m map[string]any) {
			delete(m, "h3")
			delete(m, "h4")
		}, ""},
		{"jmin above jmax", func(m map[string]any) { m["jmin"], m["jmax"] = 90, 40 }, "Jmin"},
		{"jc negative", func(m map[string]any) { m["jc"] = -1 }, "Jc"},
		{"jc above limit", func(m map[string]any) { m["jc"] = 129 }, "Jc"},
		{"jmax above limit", func(m map[string]any) { m["jmax"] = 1281 }, "Jmin/Jmax"},
		{"s1 plus 56 equals s2", func(m map[string]any) { m["s1"], m["s2"] = 15, 71 }, "equal packet sizes"},
		{"s1 zero s2 56 collision", func(m map[string]any) { m["s1"], m["s2"] = 0, 56 }, "equal packet sizes"},
		{"init equals cookie size", func(m map[string]any) {
			// 148 + s1 == 64 + s3
			m["s1"], m["s3"] = 10, 94
		}, "equal packet sizes"},
		{"s negative", func(m map[string]any) { m["s2"] = -3 }, "S2"},
		{"s above limit", func(m map[string]any) { m["s4"] = 1281 }, "S4"},
		{"bad header text", func(m map[string]any) { m["h2"] = "12ab" }, "H2"},
		{"reversed header range", func(m map[string]any) { m["h3"] = "500-100" }, "H3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			amnezia := map[string]any{}
			for k, v := range valid {
				amnezia[k] = v
			}
			tc.mutate(amnezia)
			err := ValidateAmneziaOptions(amneziaOptions(t, amnezia))
			if tc.errChunk == "" {
				if err != nil {
					t.Fatalf("expected valid, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errChunk)
			}
			if !strings.Contains(err.Error(), tc.errChunk) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.errChunk)
			}
		})
	}
}

func TestValidateAmneziaOptionsWithoutAmneziaSection(t *testing.T) {
	for _, options := range []json.RawMessage{
		nil,
		json.RawMessage(`{}`),
		json.RawMessage(`{"address":["10.0.0.1/24"],"peers":[]}`),
		json.RawMessage(`{"amnezia":null}`),
	} {
		if err := ValidateAmneziaOptions(options); err != nil {
			t.Fatalf("options %s: unexpected error %v", string(options), err)
		}
	}
}

func TestValidateAmneziaOptionsMalformedJSON(t *testing.T) {
	if err := ValidateAmneziaOptions(json.RawMessage(`{"amnezia":{"jc":"not-a-number"}}`)); err == nil {
		t.Fatal("expected error for malformed amnezia options")
	}
}

func TestGenerateAmneziaParamsProperties(t *testing.T) {
	for i := 0; i < 1000; i++ {
		params, err := GenerateAmneziaParams(i%2 == 0)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		headers := []string{params.H1, params.H2, params.H3, params.H4}
		parsed := make([]awgHeaderRange, 0, 4)
		for idx, h := range headers {
			r, err := parseAWGHeaderValue(fmt.Sprintf("H%d", idx+1), h)
			if err != nil || r == nil {
				t.Fatalf("iteration %d: generated H%d=%q invalid: %v", i, idx+1, h, err)
			}
			if r.from <= awgHeaderReservedMax {
				t.Fatalf("iteration %d: H%d=%q touches reserved values", i, idx+1, h)
			}
			if r.to > awgGenHeaderMax {
				t.Fatalf("iteration %d: H%d=%q exceeds int32 bound", i, idx+1, h)
			}
			parsed = append(parsed, *r)
		}
		for a := 0; a < len(parsed); a++ {
			for b := a + 1; b < len(parsed); b++ {
				if parsed[a].overlaps(parsed[b]) {
					t.Fatalf("iteration %d: H%d=%q overlaps H%d=%q", i, a+1, headers[a], b+1, headers[b])
				}
			}
		}
		if i%2 == 0 {
			if params.JC < 3 || params.JC > 6 {
				t.Fatalf("iteration %d: Jc=%d outside 3-6", i, params.JC)
			}
			if params.JMin < 40 || params.JMin > 89 {
				t.Fatalf("iteration %d: Jmin=%d outside 40-89", i, params.JMin)
			}
			if params.JMax < params.JMin+50 || params.JMax > params.JMin+250 {
				t.Fatalf("iteration %d: Jmax=%d outside Jmin+50..Jmin+250 (Jmin=%d)", i, params.JMax, params.JMin)
			}
		} else if params.JC != 0 || params.JMin != 0 || params.JMax != 0 {
			t.Fatalf("iteration %d: junk fields generated without includeJunk: %+v", i, params)
		}
	}
}

func TestGenerateAmneziaParamsPassValidator(t *testing.T) {
	for i := 0; i < 200; i++ {
		params, err := GenerateAmneziaParams(true)
		if err != nil {
			t.Fatal(err)
		}
		amnezia := map[string]any{
			"jc": params.JC, "jmin": params.JMin, "jmax": params.JMax,
			"s1": 15, "s2": 20, "s3": 12, "s4": 8,
			"h1": params.H1, "h2": params.H2, "h3": params.H3, "h4": params.H4,
		}
		if err := ValidateAmneziaOptions(amneziaOptions(t, amnezia)); err != nil {
			t.Fatalf("iteration %d: generated params rejected by validator: %v", i, err)
		}
	}
}

// Frontend preset catalog values (frontend/src/components/presets/
// amneziaPresets.ts) must pass the server-side validator: a preset that the
// server rejects would be a dead button in the UI.
func TestAmneziaPresetValuesPassValidator(t *testing.T) {
	presets := map[string]struct{ jc, jmin, jmax int }{
		"balanced": {jc: 4, jmin: 40, jmax: 90},
		"mobile":   {jc: 3, jmin: 40, jmax: 70},
	}
	for name, junk := range presets {
		t.Run(name, func(t *testing.T) {
			amnezia := map[string]any{
				"jc": junk.jc, "jmin": junk.jmin, "jmax": junk.jmax,
				"s1": 15, "s2": 20, "s3": 12, "s4": 8,
				"h1": "1000-1099", "h2": 2000, "h3": "3000-3099", "h4": "4000-4099",
			}
			if err := ValidateAmneziaOptions(amneziaOptions(t, amnezia)); err != nil {
				t.Fatalf("preset %s rejected by server validator: %v", name, err)
			}
		})
	}
}

func TestAWGCryptoRandIntBounds(t *testing.T) {
	if _, err := awgCryptoRandInt(10, 5); err == nil {
		t.Fatal("expected error for inverted range")
	}
	seen := map[uint64]bool{}
	for i := 0; i < 200; i++ {
		v, err := awgCryptoRandInt(3, 6)
		if err != nil {
			t.Fatal(err)
		}
		if v < 3 || v > 6 {
			t.Fatalf("value %d outside [3,6]", v)
		}
		seen[v] = true
	}
	if len(seen) < 2 {
		t.Fatalf("no variability in random values: %v", seen)
	}
	// Degenerate single-value range must work (used by the shuffle).
	v, err := awgCryptoRandInt(7, 7)
	if err != nil || v != 7 {
		t.Fatalf("single-value range: v=%d err=%v", v, err)
	}
	_ = strconv.IntSize
}
