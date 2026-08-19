package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestInboundGetAllLoadsUsersForManyInboundsInBatch(t *testing.T) {
	initSettingTestDB(t)

	inbounds := make([]model.Inbound, 0, 50)
	for i := 0; i < 50; i++ {
		inbounds = append(inbounds, model.Inbound{
			Type:    "vmess",
			Tag:     fmt.Sprintf("vmess-%02d", i),
			Options: json.RawMessage(`{}`),
		})
	}
	if err := database.GetDB().Create(&inbounds).Error; err != nil {
		t.Fatal(err)
	}

	inboundIDs := make([]uint, 0, len(inbounds))
	expectedIDs := make(map[uint]bool, len(inbounds))
	for _, inbound := range inbounds {
		inboundIDs = append(inboundIDs, inbound.Id)
		expectedIDs[inbound.Id] = true
	}
	inboundIDsJSON, err := json.Marshal(inboundIDs)
	if err != nil {
		t.Fatal(err)
	}

	clients := make([]model.Client, 0, 100)
	for i := 0; i < 100; i++ {
		clients = append(clients, model.Client{
			Name:     fmt.Sprintf("client-%03d", i),
			Inbounds: json.RawMessage(inboundIDsJSON),
		})
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	got, err := (&InboundService{}).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	seen := 0
	for _, inbound := range *got {
		id, ok := inbound["id"].(uint)
		if !ok || !expectedIDs[id] {
			continue
		}
		users, ok := inbound["users"].([]string)
		if !ok {
			t.Fatalf("inbound %d users has unexpected type %T", id, inbound["users"])
		}
		if len(users) != 100 {
			t.Fatalf("inbound %d expected 100 users, got %d", id, len(users))
		}
		if users[0] != "client-000" || users[99] != "client-099" {
			t.Fatalf("inbound %d users are not in client order: first=%q last=%q", id, users[0], users[99])
		}
		seen++
	}
	if seen != 50 {
		t.Fatalf("expected 50 tested inbounds, got %d", seen)
	}
}

// TestInboundGetAllEmitsUsersForSudoku proves a keyless sudoku inbound is
// client-assignable: GetAll must emit a `users` key with the names of assigned
// clients in client-id order (issue #4 — frontend filters the assignment
// selector on the presence of `users`).
func TestInboundGetAllEmitsUsersForSudoku(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "sudoku",
		Tag:     "sudoku-1",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inboundIDs := json.RawMessage(fmt.Sprintf("[%d]", inbound.Id))
	clients := []model.Client{
		{Enable: true, Name: "alice", Inbounds: inboundIDs},
		{Enable: true, Name: "bob", Inbounds: inboundIDs},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	got, err := (&InboundService{}).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range *got {
		if item["id"] != inbound.Id {
			continue
		}
		found = true
		users, ok := item["users"].([]string)
		if !ok {
			t.Fatalf("sudoku inbound must expose a users list, got %T (%v)", item["users"], item["users"])
		}
		if len(users) != 2 || users[0] != "alice" || users[1] != "bob" {
			t.Fatalf("sudoku users must list assigned clients in client order, got %v", users)
		}
	}
	if !found {
		t.Fatal("sudoku inbound missing from GetAll output")
	}
}

// TestInboundGetAllOmitsUsersForNonAssignableTypes is the regression guard for
// the sudoku fix: managed shadowsocks, shadowtls v<3 and transparent/local
// types (direct/tun/bond) must still have NO `users` key.
func TestInboundGetAllOmitsUsersForNonAssignableTypes(t *testing.T) {
	initSettingTestDB(t)

	inbounds := []model.Inbound{
		{Type: "shadowsocks", Tag: "ss-managed", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8388,"managed":true,"method":"2022-blake3-aes-128-gcm"}`)},
		{Type: "shadowtls", Tag: "stls-v2", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8444,"version":2}`)},
		{Type: "direct", Tag: "direct-1", Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8445}`)},
		{Type: "tun", Tag: "tun-1", Options: json.RawMessage(`{}`)},
		{Type: "bond", Tag: "bond-1", Options: json.RawMessage(`{}`)},
	}
	if err := database.GetDB().Create(&inbounds).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]uint, 0, len(inbounds))
	for _, in := range inbounds {
		ids = append(ids, in.Id)
	}
	idsJSON, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Client{
		Enable:   true,
		Name:     "alice",
		Inbounds: json.RawMessage(idsJSON),
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := (&InboundService{}).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	seen := 0
	for _, item := range *got {
		for _, in := range inbounds {
			if item["id"] != in.Id {
				continue
			}
			seen++
			if _, has := item["users"]; has {
				t.Errorf("%s inbound %q must NOT expose users, got %v", in.Type, in.Tag, item["users"])
			}
		}
	}
	if seen != len(inbounds) {
		t.Fatalf("expected %d tested inbounds, saw %d", len(inbounds), seen)
	}
}

// TestAddUsersLeavesSudokuCoreConfigUntouched proves the core-config generation
// path is unaffected by the assignability fix: the sing-box sudoku inbound has
// no `users` option, so addUsers must return the JSON unchanged even when
// clients are assigned to the inbound.
func TestAddUsersLeavesSudokuCoreConfigUntouched(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "sudoku",
		Tag:     "sudoku-core",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY"}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Client{
		Enable:   true,
		Name:     "alice",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
	}).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON := []byte(`{"type":"sudoku","tag":"sudoku-core","listen":"0.0.0.0","listen_port":8443,"key":"SHARED-KEY"}`)
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "sudoku")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(inboundJSON) {
		t.Fatalf("addUsers must not modify a sudoku core config.\n got: %s\nwant: %s", out, inboundJSON)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if _, has := got["users"]; has {
		t.Fatalf("sudoku core config must never contain users: %s", out)
	}
}

func TestAddUsersDropsTrojanTopLevelPassword(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "trojan",
		Tag:     "trojan-1",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443,"password":"hello","network":["tcp"]}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Client{
		Name:     "alice",
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
	}).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON, err := json.Marshal(map[string]any{
		"type": "trojan", "tag": "trojan-1",
		"listen": "0.0.0.0", "listen_port": 443, "password": "hello", "network": []string{"tcp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "trojan")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	// sing-box's Trojan inbound has no top-level "password" or "network"
	// fields (only "users"), and rejects the whole config if either is present.
	for _, key := range []string{"password", "network"} {
		if _, has := got[key]; has {
			t.Errorf("trojan inbound must not keep top-level %s: %s", key, out)
		}
	}
	if _, has := got["users"]; !has {
		t.Errorf("trojan inbound should have users injected: %s", out)
	}
}

// TestAddUsersStripsOutboundOnlyTrustTunnelKeys is the regression guard for the
// inbound core-start crash: the pre-fix TrustTunnel UI exposed an unguarded
// QUIC switch and initialized multiplex unconditionally, so an inbound save
// could emit `quic` or `multiplex` at top level. Username/password/health_check
// were already out-gated; stripping them too is defensive cleanup for legacy
// or outbound-shaped payloads. The fork's TrustTunnel inbound struct rejects
// these fields, while network/congestion_controller/cwnd remain valid.
func TestAddUsersStripsOutboundOnlyTrustTunnelKeys(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "trusttunnel",
		Tag:     "tt",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":443}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Create(&model.Client{
		Enable:   true,
		Name:     "alice",
		Config:   json.RawMessage(`{"trusttunnel":{"name":"alice","password":"pw1"}}`),
		Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
	}).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON := []byte(`{"type":"trusttunnel","tag":"tt","listen":"0.0.0.0","listen_port":443,"quic":true,"health_check":true,"username":"u","password":"p","multiplex":{},"network":["tcp","udp"],"congestion_controller":"bbr","cwnd":32}`)
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "trusttunnel")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"quic", "health_check", "username", "password", "multiplex"} {
		if _, has := got[key]; has {
			t.Errorf("trusttunnel inbound must not keep outbound-only key %q: %s", key, out)
		}
	}
	for _, key := range []string{"network", "congestion_controller", "cwnd"} {
		if _, has := got[key]; !has {
			t.Errorf("trusttunnel inbound must keep inbound key %q: %s", key, out)
		}
	}
	users, _ := got["users"].([]any)
	if len(users) == 0 {
		t.Fatalf("trusttunnel inbound should have users injected: %s", out)
	}
}

func TestInboundCoreJSONStripsKeylessInboundOnlyOptions(t *testing.T) {
	initSettingTestDB(t)

	for _, tc := range []struct {
		name        string
		inboundType string
		options     string
		forbidden   []string
		preserved   []string
	}{
		{
			name:        "sudoku",
			inboundType: "sudoku",
			options:     `{"listen_port":443,"key":"public","master_key":"secret","http_mask":{},"disable_http_mask":false,"http_mask_mode":"auto","path_root":"/mask","fallback":"https://example.com"}`,
			forbidden:   []string{"master_key", "http_mask"},
			preserved:   []string{"listen_port", "key", "disable_http_mask", "http_mask_mode", "path_root", "fallback"},
		},
		{
			name:        "call",
			inboundType: "call",
			options:     `{"listen":"0.0.0.0","listen_port":443,"udp_timeout":30,"proxy_protocol":true,"proxy_protocol_accept_no_header":true,"join_link":"https://example.com/join","platform":"dion","cookies":[{"name":"sid","value":"x"}]}`,
			forbidden:   []string{"listen", "listen_port", "udp_timeout", "proxy_protocol", "proxy_protocol_accept_no_header"},
			preserved:   []string{"join_link", "platform", "cookies"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := model.Inbound{
				Type:    tc.inboundType,
				Tag:     tc.name + "-core",
				Options: json.RawMessage(tc.options),
			}
			data, err := inboundCoreJSON(&InboundService{}, database.GetDB(), row)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			for _, key := range tc.forbidden {
				if _, ok := got[key]; ok {
					t.Errorf("%s inbound leaked core-incompatible key %q: %s", tc.inboundType, key, data)
				}
			}
			for _, key := range tc.preserved {
				if _, ok := got[key]; !ok {
					t.Errorf("%s inbound dropped valid key %q: %s", tc.inboundType, key, data)
				}
			}
		})
	}
}

func TestSanitizeInboundJSONForCorePreservesUntouchedBytes(t *testing.T) {
	input := []byte(`{"type":"call","tag":"plain","join_link":"https://example.com/join"}`)
	output, err := sanitizeInboundJSONForCore("call", input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, input) {
		t.Fatalf("untouched inbound JSON was rewritten: got %s, want %s", output, input)
	}
}

func TestAddUsersInjectsExtendedProtocolUsers(t *testing.T) {
	for _, tc := range []struct {
		inboundType string
		userConfig  string
		wantName    string
		wantSecret  string
		secretKey   string
	}{
		{inboundType: "mieru", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "trusttunnel", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "ssh", userConfig: `{"name":"alice","password":"pw1"}`, wantName: "alice", wantSecret: "pw1", secretKey: "password"},
		{inboundType: "mtproxy", userConfig: `{"name":"alice","secret":"0123456789abcdef0123456789abcdef"}`, wantName: "alice", wantSecret: "0123456789abcdef0123456789abcdef", secretKey: "secret"},
	} {
		t.Run(tc.inboundType, func(t *testing.T) {
			initSettingTestDB(t)

			inbound := model.Inbound{
				Type:    tc.inboundType,
				Tag:     tc.inboundType + "-1",
				Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":2999}`),
			}
			if err := database.GetDB().Create(&inbound).Error; err != nil {
				t.Fatal(err)
			}
			config := json.RawMessage(fmt.Sprintf(`{"%s":%s}`, tc.inboundType, tc.userConfig))
			if err := database.GetDB().Create(&model.Client{
				Enable:   true,
				Name:     "alice",
				Config:   config,
				Inbounds: json.RawMessage(fmt.Sprintf("[%d]", inbound.Id)),
			}).Error; err != nil {
				t.Fatal(err)
			}

			inboundJSON, err := json.Marshal(map[string]any{
				"type": tc.inboundType, "tag": tc.inboundType + "-1", "listen": "0.0.0.0", "listen_port": 2999,
			})
			if err != nil {
				t.Fatal(err)
			}
			out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, tc.inboundType)
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Users []map[string]string `json:"users"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			users := got.Users
			if len(users) != 1 || users[0]["name"] != tc.wantName || users[0][tc.secretKey] != tc.wantSecret {
				t.Fatalf("%s users were not injected correctly: %s", tc.inboundType, out)
			}
		})
	}
}

func TestFetchUsersByConditionRejectsUnsupportedInboundTypeBeforeSQL(t *testing.T) {
	_, err := (&InboundService{}).fetchUsersByCondition(nil, "vmess'); DROP TABLE clients; --", "1=1", map[string]interface{}{})
	if err == nil {
		t.Fatal("unsupported inbound type should be rejected before SQL execution")
	}
}

func TestFetchUsersByConditionRejectsUnexpectedJSONFieldBeforeSQL(t *testing.T) {
	const inboundType = "test-malicious-field"
	old, existed := userJSONField[inboundType]
	userJSONField[inboundType] = "vmess') FROM clients; --"
	t.Cleanup(func() {
		if existed {
			userJSONField[inboundType] = old
		} else {
			delete(userJSONField, inboundType)
		}
	})

	_, err := (&InboundService{}).fetchUsersByCondition(nil, inboundType, "1=1", map[string]interface{}{})
	if err == nil {
		t.Fatal("unexpected JSON field should be rejected before SQL execution")
	}
}

func TestAddUsersSkipsClientsMissingProtocolConfig(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "mieru",
		Tag:     "mieru-shared",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":2999}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inboundIDs := json.RawMessage(fmt.Sprintf("[%d]", inbound.Id))

	clients := []model.Client{
		{
			Enable:   true,
			Name:     "alice",
			Config:   json.RawMessage(`{"mieru":{"name":"alice","password":"pw1"}}`),
			Inbounds: inboundIDs,
		},
		{
			Enable: true,
			Name:   "bob",
			// bob only has a vmess account, so he should be ignored when building
			// the mieru inbound users list rather than causing a NULL scan error.
			Config:   json.RawMessage(`{"vmess":{"id":"a0b1c2d3-e4f5-6789-0123-456789abcdef"}}`),
			Inbounds: inboundIDs,
		},
		{
			Enable:   true,
			Name:     "carol",
			Config:   nil,
			Inbounds: inboundIDs,
		},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON, err := json.Marshal(map[string]any{
		"type": "mieru", "tag": "mieru-shared", "listen": "0.0.0.0", "listen_port": 2999,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "mieru")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Users []map[string]string `json:"users"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Users) != 1 || got.Users[0]["name"] != "alice" {
		t.Fatalf("expected only alice in mieru users, got %s", out)
	}
}

func TestAddUsersKeepsUsersArrayWhenAllProtocolConfigsAreMissing(t *testing.T) {
	initSettingTestDB(t)

	inbound := model.Inbound{
		Type:    "mieru",
		Tag:     "mieru-empty",
		Options: json.RawMessage(`{"listen":"0.0.0.0","listen_port":2999}`),
	}
	if err := database.GetDB().Create(&inbound).Error; err != nil {
		t.Fatal(err)
	}
	inboundIDs := json.RawMessage(fmt.Sprintf("[%d]", inbound.Id))
	clients := []model.Client{
		{
			Enable:   true,
			Name:     "bob",
			Config:   json.RawMessage(`{"vmess":{"id":"a0b1c2d3-e4f5-6789-0123-456789abcdef"}}`),
			Inbounds: inboundIDs,
		},
		{
			Enable:   true,
			Name:     "carol",
			Config:   nil,
			Inbounds: inboundIDs,
		},
	}
	if err := database.GetDB().Create(&clients).Error; err != nil {
		t.Fatal(err)
	}

	inboundJSON, err := json.Marshal(map[string]any{
		"type": "mieru", "tag": "mieru-empty", "listen": "0.0.0.0", "listen_port": 2999,
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := (&InboundService{}).addUsers(database.GetDB(), inboundJSON, inbound.Id, "mieru")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Users []map[string]string `json:"users"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.Users == nil || len(got.Users) != 0 {
		t.Fatalf("expected an empty users array, got %s", out)
	}
}
