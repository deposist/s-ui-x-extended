package service

import (
	"encoding/json"
	"testing"
)

func TestDecodeClientCredentialsMapsPanelFields(t *testing.T) {
	const config = `{"vless":{"uuid":"11111111-1111-4111-8111-111111111111","flow":"xtls-rprx-vision"},"vmess":{"uuid":"22222222-2222-4222-8222-222222222222","alterId":0},"hysteria":{"auth_str":"auth"},"mtproxy":{"secret":"secret"},"ssh":{"password":"pw","authorizedKeys":["ssh-ed25519 AAA"]}}`
	got, err := decodeClientCredentials(json.RawMessage(config))
	if err != nil {
		t.Fatalf("decode credentials: %v", err)
	}
	for inboundType, want := range map[string]fleetCredential{
		"vless":    {uuid: "11111111-1111-4111-8111-111111111111", flow: "xtls-rprx-vision"},
		"vmess":    {uuid: "22222222-2222-4222-8222-222222222222", alterID: 0},
		"hysteria": {password: "auth"},
		"mtproxy":  {secret: "secret"},
		"ssh":      {password: "pw", authorizedKeys: []string{"ssh-ed25519 AAA"}},
	} {
		credential, ok := got.forType(inboundType)
		if !ok || credential.uuid != want.uuid || credential.password != want.password || credential.secret != want.secret || credential.flow != want.flow || credential.alterID != want.alterID || len(credential.authorizedKeys) != len(want.authorizedKeys) {
			t.Errorf("credential for %s = %+v, present %v; want %+v", inboundType, credential, ok, want)
			continue
		}
		for i := range want.authorizedKeys {
			if credential.authorizedKeys[i] != want.authorizedKeys[i] {
				t.Errorf("authorized keys for %s = %v; want %v", inboundType, credential.authorizedKeys, want.authorizedKeys)
			}
		}
	}
	if _, ok := got.forType("shadowsocks"); ok {
		t.Fatal("manager-unsupported Shadowsocks users must not be projected")
	}
}


func TestDecodeClientCredentialsPreservesProtocolUsernames(t *testing.T) {
	got, err := decodeClientCredentials(json.RawMessage(`{"mixed":{"username":"auth-user"},"socks":{"username":"socks-user"},"vless":{"uuid":"11111111-1111-4111-8111-111111111111"}}`))
	if err != nil { t.Fatal(err) }
	if got["mixed"].username != "auth-user" || got["socks"].username != "socks-user" {
		t.Fatalf("protocol auth usernames were lost: mixed=%q socks=%q", got["mixed"].username, got["socks"].username)
	}
	if got["vless"].username != "" { t.Fatalf("UUID protocol must not acquire synthetic username: %q", got["vless"].username) }
}

func TestDecodeClientCredentialsRejectsMalformedJSON(t *testing.T) {
	if _, err := decodeClientCredentials(json.RawMessage("{")); err == nil {
		t.Fatal("malformed credential JSON must be rejected")
	}
}
