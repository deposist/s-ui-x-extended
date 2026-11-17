package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

func TestInboundSavePreservesNestedGroupMember(t *testing.T) {
	for _, groupType := range []string{"bond", CoreFailoverType} {
		for _, action := range []string{"del", "edit"} {
			t.Run(groupType+"/"+action, func(t *testing.T) {
				initSettingTestDB(t)
				db := database.GetDB()
				member := model.Inbound{Type: "direct", Tag: "nested-member", Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`)}
				if err := db.Create(&member).Error; err != nil {
					t.Fatal(err)
				}
				group := model.Inbound{Type: groupType, Tag: "nested-group", Options: json.RawMessage(`{"inbounds":["nested-member"]}`)}
				if err := db.Create(&group).Error; err != nil {
					t.Fatal(err)
				}
				payload := json.RawMessage(`"nested-member"`)
				if action == "edit" {
					payload = json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"direct","tag":"renamed-member","listen":"127.0.0.1","listen_port":0}`, member.Id))
				}
				if _, err := (&InboundService{}).Save(db, action, payload, "", "example.com"); err == nil {
					t.Fatal("nested group member must not be deleted or renamed while referenced")
				}
				var saved model.Inbound
				if err := db.First(&saved, member.Id).Error; err != nil || saved.Tag != member.Tag {
					t.Fatalf("referenced member was changed: tag=%q err=%v", saved.Tag, err)
				}
			})
		}
	}
}

func TestConfigSaveNestedInboundMemberRebuildsParent(t *testing.T) {
	initSettingTestDB(t)
	db := database.GetDB()
	member := model.Inbound{Type: "direct", Tag: "nested-member", Options: json.RawMessage(`{"listen":"127.0.0.1","listen_port":0}`)}
	if err := db.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	group := model.Inbound{Type: "bond", Tag: "nested-group", Options: json.RawMessage(`{"inbounds":["nested-member"]}`)}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	instance := startTestCore(t)
	configService := NewConfigServiceWithRuntime(NewRuntime(instance))
	if err := configService.RestartCore(); err != nil {
		t.Fatal(err)
	}
	before := instance.GetInstance()
	recorder := &inboundOpsRecorder{}
	recorder.stubInboundHooks(t)
	payload := json.RawMessage(fmt.Sprintf(`{"id":%d,"type":"direct","tag":"nested-member","listen":"127.0.0.1","listen_port":0,"override_address":"127.0.0.2"}`, member.Id))
	if _, err := configService.Save("inbounds", "edit", payload, "", "admin", "example.com"); err != nil {
		t.Fatal(err)
	}
	if len(recorder.ops) != 0 || instance.GetInstance() == before {
		t.Fatalf("editing a nested member left its parent unchanged: partial ops=%v", recorder.ops)
	}
	if err := instance.RemoveInbound(group.Tag); err != nil {
		t.Fatalf("nested group missing after rebuild: %v", err)
	}
}
