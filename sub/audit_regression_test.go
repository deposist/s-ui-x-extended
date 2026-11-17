package sub

import (
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
	"gorm.io/gorm"
)

func TestSubServerStopReleasesUnservedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	address := listener.Addr().String()
	server := NewServer()
	server.listener = listener
	server.httpServer = &http.Server{}
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	restarted, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listener remains bound after Stop before Serve: %v", err)
	}
	_ = restarted.Close()
}

func TestSubServerRepeatedStartStop(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"subListen": "127.0.0.1", "subPort": "0"} {
		if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer()
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})
	for i := 0; i < 25; i++ {
		if err := server.Start(); err != nil {
			t.Fatal(err)
		}
		if err := server.Stop(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSubscriptionLookupRejectsUnreadableSecretPolicy(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", "subSecretRequired").Update("value", "invalid").Error; err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: true, Name: "legacy", SubSecret: "private-id", Inbounds: []byte("[]"), Links: []byte("[]")}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if got, err := (&SubService{}).getClientBySubId(client.Name); err == nil || got != nil {
		t.Fatal("unreadable security policy allowed a legacy name lookup")
	}
}

func TestJSONAddressTLSOverridesAreIndependent(t *testing.T) {
	inbound := &model.Inbound{
		Id:      1,
		Options: []byte(`{}`),
		OutJson: []byte(`{"type":"trojan","tag":"server","tls":{"enabled":true,"server_name":"base.example"}}`),
		Addrs:   []byte(`[{"server":"one.example","server_port":443,"tls":{"server_name":"one.example"}},{"server":"two.example","server_port":443,"tls":{"server_name":"two.example"}}]`),
	}
	outbounds, _, err := (&JsonService{}).getOutbounds(json.RawMessage(`{"trojan":{"password":"test"}}`), []*model.Inbound{inbound})
	if err != nil {
		t.Fatal(err)
	}
	if len(*outbounds) != 2 {
		t.Fatalf("got %d outbounds, want 2", len(*outbounds))
	}
	for i, want := range []string{"one.example", "two.example"} {
		tls := (*outbounds)[i]["tls"].(map[string]interface{})
		if got := tls["server_name"]; got != want {
			t.Fatalf("outbound %d TLS server_name = %v, want %s", i, got, want)
		}
	}
}

func TestRevokedSubscriptionCannotRepopulateOutputCache(t *testing.T) {
	initSubTestDB(t)
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: true, Name: "cache-client", SubSecret: "cache-secret", Inbounds: []byte("[]"), Links: []byte("[]")}
	db := database.GetDB()
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	loaded := make(chan struct{})
	resume := make(chan struct{})
	var blocked atomic.Bool
	callback := "test:block-loaded-subscription"
	if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "clients" && blocked.CompareAndSwap(false, true) {
			close(loaded)
			<-resume
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Query().Remove(callback); err != nil {
			t.Error(err)
		}
	})
	finished := make(chan error, 1)
	go func() { _, _, err := (&SubService{}).GetSubs(client.SubSecret); finished <- err }()
	select {
	case <-loaded:
	case <-time.After(5 * time.Second):
		close(resume)
		t.Fatal("subscription did not reach the loaded-client boundary")
	}
	if err := db.Model(model.Client{}).Where("id = ?", client.Id).Update("enable", false).Error; err != nil {
		close(resume)
		t.Fatal(err)
	}
	InvalidateClientSubscriptionOutputCache([]string{client.SubSecret, client.Name})
	close(resume)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight subscription did not finish")
	}
	if body, _, err := (&SubService{}).GetSubs(client.SubSecret); err == nil || body != nil {
		t.Fatal("a revoked client was served from a repopulated output cache")
	}
}
