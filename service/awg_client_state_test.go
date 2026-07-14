package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

type recordingAWGClientStateHook struct {
	suspended [][]uint
	resumed   []uint
}

func (h *recordingAWGClientStateHook) SuspendClients(_ context.Context, ids []uint) error {
	h.suspended = append(h.suspended, append([]uint(nil), ids...))
	return nil
}

func (h *recordingAWGClientStateHook) ResumeClient(_ context.Context, id uint) error {
	h.resumed = append(h.resumed, id)
	return nil
}

func TestDepleteClientsNotifiesAWGAfterCommit(t *testing.T) {
	initSettingTestDB(t)
	hook := &recordingAWGClientStateHook{}
	runtime := NewRuntimeWithCoreProvider(nil)
	runtime.SetAWGClientStateHook(hook)
	client := model.Client{Enable: true, Name: "deplete-awg", Inbounds: json.RawMessage(`[]`), Links: json.RawMessage(`[]`), Volume: 1, Up: 1}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := (&ClientService{Runtime: runtime}).DepleteClients(); err != nil {
		t.Fatal(err)
	}
	if len(hook.suspended) != 1 || len(hook.suspended[0]) != 1 || hook.suspended[0][0] != client.Id {
		t.Fatalf("suspended=%v", hook.suspended)
	}
	var stored model.Client
	if err := database.GetDB().First(&stored, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Enable {
		t.Fatal("hook ran without committed inactive state")
	}
}
