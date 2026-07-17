package util

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

const (
	testSudokuInboundKey = "inbound-public-or-legacy-key"
	testSudokuClientKeyA = "01000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000"
	testSudokuClientKeyB = "02000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000"
)

func decodeSudokuPayload(t *testing.T, uri string) map[string]any {
	t.Helper()
	if !strings.HasPrefix(uri, "sudoku://") {
		t.Fatalf("not sudoku URI: %s", uri)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(uri, "sudoku://"))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func sudokuTestInbound() *model.Inbound {
	return &model.Inbound{Type: "sudoku", Tag: "sudoku-test", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443,"key":"` + testSudokuInboundKey + `","aead_method":"chacha20-poly1305","table_type":"ascii","path_root":"/mask"}`), Addrs: json.RawMessage(`[]`)}
}

func TestSudokuLinkUsesPerClientKeyAndPreservesInboundFields(t *testing.T) {
	in := sudokuTestInbound()
	linkA := LinkGenerator(json.RawMessage(`{"sudoku":{"key":"`+testSudokuClientKeyA+`","server":"evil","aead_method":"none","junk":true}}`), in, "example.com")
	linkB := LinkGenerator(json.RawMessage(`{"sudoku":{"key":"`+testSudokuClientKeyB+`"}}`), in, "example.com")
	if len(linkA) != 1 || len(linkB) != 1 || linkA[0] == linkB[0] {
		t.Fatalf("distinct clients did not get distinct links: %v %v", linkA, linkB)
	}

	a := decodeSudokuPayload(t, linkA[0])
	b := decodeSudokuPayload(t, linkB[0])
	if a["k"] != testSudokuClientKeyA || b["k"] != testSudokuClientKeyB {
		t.Fatalf("wrong personal keys: A=%v B=%v", a, b)
	}
	if a["h"] != "example.com" || a["p"] != float64(443) || a["e"] != "chacha20-poly1305" || a["a"] != "ascii" {
		t.Fatalf("client config overrode inbound fields: %v", a)
	}
	if _, ok := a["junk"]; ok {
		t.Fatalf("arbitrary client field leaked: %v", a)
	}
}

func TestSudokuLinkSelectsKeyForInboundID(t *testing.T) {
	in := sudokuTestInbound()
	in.Id = 42
	config := json.RawMessage(`{"sudoku":{"keys":{"42":"` + testSudokuClientKeyA + `","99":"` + testSudokuClientKeyB + `"}}}`)
	links := LinkGenerator(config, in, "example.com")
	if len(links) != 1 {
		t.Fatalf("expected one link, got %v", links)
	}
	if got := decodeSudokuPayload(t, links[0])["k"]; got != testSudokuClientKeyA {
		t.Fatalf("selected key = %v, want inbound-specific key", got)
	}
}

func TestSudokuLinkFallbackIsBackwardCompatible(t *testing.T) {
	in := sudokuTestInbound()
	for _, config := range []json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{"sudoku":{}}`), json.RawMessage(`{"sudoku":{"key":""}}`)} {
		links := LinkGenerator(config, in, "example.com")
		if len(links) != 1 || decodeSudokuPayload(t, links[0])["k"] != testSudokuInboundKey {
			t.Fatalf("fallback failed for %s: %v", config, links)
		}
	}
}
