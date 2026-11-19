package service

import (
	"reflect"
	"testing"
)

func TestPeerProtocolCredentialsReachProxyUserSchema(t *testing.T) {
	const id = "c46a4d36-c8f2-4aa7-828a-8df136220995"
	for _, test := range []struct {
		kind string
		id   string
		pass string
		auth string
		want map[string]string
	}{
		{"anytls", "", "test-password", "", map[string]string{"name": "alice", "password": "test-password"}},
		{"hysteria", "", "", "test-auth", map[string]string{"name": "alice", "auth_str": "test-auth"}},
		{"hysteria", "", "test-password", "", nil},
		{"hysteria2", "", "test-password", "", map[string]string{"name": "alice", "password": "test-password"}},
		{"tuic", id, "test-password", "", map[string]string{"name": "alice", "uuid": id, "password": "test-password"}},
		{"tuic", "", "test-password", "", nil},
		{"hysteria2", id, "test-password", "", nil},
		{"shadowsocks", "", "test-password", "", nil},
	} {
		t.Run(test.kind+"/"+test.id, func(t *testing.T) {
			user := peerUser{Type: test.kind, Name: "alice", UUID: test.id}
			user.Password, user.AuthString = test.pass, test.auth
			if validPeerProtocolCredential(user) != (test.want != nil) {
				t.Fatal("unexpected credential validation result")
			}
			if test.want != nil && (!peerProtocolSupported(test.kind) || !reflect.DeepEqual(peerInboundCredential(user, nil), test.want)) {
				t.Fatal("credential did not render in the proxy's user schema")
			}
		})
	}
}

func TestPeerCredentialIdentityMatchesAuthentication(t *testing.T) {
	const id = "c46a4d36-c8f2-4aa7-828a-8df136220995"
	for _, test := range []struct {
		user peerUser
		want string
	}{
		{peerUser{Type: "vless", UUID: "C46A4D36-C8F2-4AA7-828A-8DF136220995"}, id},
		{peerUser{Type: "vmess", UUID: "{c46a4d36-c8f2-4aa7-828a-8df136220995}"}, id},
		{peerUser{Type: "tuic", UUID: id, Password: "ignored-by-identity"}, id},
		{peerUser{Type: "hysteria", AuthString: "test-auth"}, "test-auth"},
		{peerUser{Type: "hysteria2", Password: "test-password"}, "test-password"},
		{peerUser{Type: "vless", UUID: "bad-uuid"}, ""},
	} {
		if got := peerCredentialIdentity(test.user); got != test.want {
			t.Errorf("%s identity did not match the authentication key", test.user.Type)
		}
	}
}
