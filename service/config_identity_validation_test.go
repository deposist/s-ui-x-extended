package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateEntityIdentity(t *testing.T) {
	cases := []struct {
		name    string
		obj     string
		act     string
		data    string
		wantErr bool
	}{
		{"outbound valid tag", "outbounds", "new", `{"tag":"proxy-a","type":"direct"}`, false},
		{"outbound empty tag", "outbounds", "new", `{"tag":"","type":"direct"}`, true},
		{"outbound whitespace tag", "outbounds", "new", `{"tag":"   ","type":"direct"}`, true},
		{"outbound missing tag", "outbounds", "new", `{"type":"direct"}`, true},
		{"outbound empty tag on edit", "outbounds", "edit", `{"id":2,"tag":"","type":"direct"}`, true},
		{"inbound empty tag", "inbounds", "new", `{"tag":"","type":"vless"}`, true},
		{"service empty tag", "services", "new", `{"tag":"","type":"derp"}`, true},
		{"endpoint empty tag", "endpoints", "new", `{"tag":"","type":"wireguard"}`, true},
		{"provider empty tag", "providers", "new", `{"tag":"","type":"remote"}`, true},
		{"tls valid name", "tls", "new", `{"name":"cert-a"}`, false},
		{"tls empty name", "tls", "new", `{"name":""}`, true},
		{"tls missing name", "tls", "new", `{"server":{}}`, true},
		{"tls tag does not satisfy name", "tls", "new", `{"tag":"cert-a"}`, true},
		{"edit blank row to real tag", "outbounds", "edit", `{"id":2,"tag":"fixed"}`, false},
		{"delete by tag string", "outbounds", "del", `"proxy-a"`, false},
		{"delete of blank tag", "outbounds", "del", `""`, false},
		{"clients not identity checked", "clients", "new", `{"name":""}`, false},
		{"settings untouched", "settings", "new", `{"webPort":"2095"}`, false},
		{"config untouched", "config", "new", `{"log":{"level":"info"}}`, false},
		{"non-object body deferred", "outbounds", "new", `["not-an-object"]`, false},
		{"non-string tag rejected", "outbounds", "new", `{"tag":123}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEntityIdentity(tc.obj, tc.act, json.RawMessage(tc.data))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %s/%s %s, got nil", tc.obj, tc.act, tc.data)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %s/%s %s: %v", tc.obj, tc.act, tc.data, err)
			}
		})
	}
}

func TestValidateEntityIdentityErrorMentionsField(t *testing.T) {
	err := validateEntityIdentity("tls", "new", json.RawMessage(`{"name":""}`))
	if err == nil {
		t.Fatal("expected an error for a blank tls name")
	}
	if !strings.Contains(err.Error(), "tls") || !strings.Contains(err.Error(), "name") {
		t.Fatalf("error should name the object and field, got: %v", err)
	}
}

func TestDispatchSaveRejectsBlankTagBeforeTouchingDB(t *testing.T) {
	s := &ConfigService{}
	_, _, _, err := s.dispatchSave(nil, "outbounds", "new", json.RawMessage(`{"tag":"","type":"direct"}`), "", "")
	if err == nil {
		t.Fatal("dispatchSave accepted a blank outbound tag")
	}
}

func TestDispatchSaveRejectsUnsafeLogOutputThroughCorePolicy(t *testing.T) {
	s := &ConfigService{}
	_, _, _, err := s.dispatchSave(nil, "config", "set", json.RawMessage(`{"log":{"output":"../../etc/passwd"}}`), "", "")
	if err == nil {
		t.Fatal("dispatchSave accepted an unsafe log.output path")
	}
}

func TestEntityIdentityField(t *testing.T) {
	cases := []struct {
		obj       string
		wantField string
		wantOK    bool
	}{
		{"outbounds", "tag", true},
		{"inbounds", "tag", true},
		{"services", "tag", true},
		{"endpoints", "tag", true},
		{"providers", "tag", true},
		{"tls", "name", true},
		{"clients", "", false},
		{"settings", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.obj, func(t *testing.T) {
			field, ok := entityIdentityField(tc.obj)
			if field != tc.wantField || ok != tc.wantOK {
				t.Fatalf("entityIdentityField(%q) = (%q, %v), want (%q, %v)", tc.obj, field, ok, tc.wantField, tc.wantOK)
			}
		})
	}
}
