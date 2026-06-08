package paidsub

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestGenerateClientConfig(t *testing.T) {
	raw, err := generateClientConfig("alice")
	if err != nil {
		t.Fatalf("generateClientConfig: %v", err)
	}
	var cfg map[string]map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, proto := range []string{"vless", "vmess", "trojan", "shadowsocks", "shadowsocks16", "socks", "http", "hysteria2", "tuic"} {
		if _, ok := cfg[proto]; !ok {
			t.Errorf("config missing protocol %q", proto)
		}
	}
	// vless/vmess must share a non-empty uuid.
	if cfg["vless"]["uuid"] == "" || cfg["vless"]["uuid"] != cfg["vmess"]["uuid"] {
		t.Errorf("vless/vmess uuid mismatch or empty")
	}
	// shadowsocks key must be 32 bytes, shadowsocks16 must be 16 bytes.
	ss32, _ := cfg["shadowsocks"]["password"].(string)
	if b, err := base64.StdEncoding.DecodeString(ss32); err != nil || len(b) != 32 {
		t.Errorf("shadowsocks key not 32 bytes b64: len=%d err=%v", len(b), err)
	}
	ss16, _ := cfg["shadowsocks16"]["password"].(string)
	if b, err := base64.StdEncoding.DecodeString(ss16); err != nil || len(b) != 16 {
		t.Errorf("shadowsocks16 key not 16 bytes b64: len=%d err=%v", len(b), err)
	}
}

func TestSanitizeUsername(t *testing.T) {
	cases := map[string]string{
		"":          "telegram",
		"alice":     "@alice",
		"bad name!": "@badname",
		"a_b_2024":  "@a_b_2024",
		"!!!":       "telegram",
		"Иван":      "telegram", // non-ascii stripped
		"john.doe":  "@johndoe",
	}
	for in, want := range cases {
		if got := sanitizeUsername(in); got != want {
			t.Errorf("sanitizeUsername(%q) = %q, want %q", in, got, want)
		}
	}
}
