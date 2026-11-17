package service

import (
	"testing"

	CM "github.com/sagernet/sing-box/service/manager/constant"
)

type fleetUserTypeManager struct {
	CM.Manager
	user CM.User
}

func (m *fleetUserTypeManager) GetUsers(map[string][]string) ([]CM.User, error) {
	return []CM.User{m.user}, nil
}

func (m *fleetUserTypeManager) UpdateUser(_ int, update CM.UserUpdate) (CM.User, error) {
	// The manager update API has no Type field; credentials alone cannot
	// convert a record to a different protocol.
	m.user.UUID = update.UUID
	m.user.Password = update.Password
	m.user.Secret = update.Secret
	m.user.AuthorizedKeys = update.AuthorizedKeys
	m.user.Flow = update.Flow
	m.user.AlterID = update.AlterID
	return m.user, nil
}

func (m *fleetUserTypeManager) DeleteUser(int) (CM.User, error) {
	previous := m.user
	m.user = CM.User{}
	return previous, nil
}

func (m *fleetUserTypeManager) CreateUser(create CM.UserCreate) (CM.User, error) {
	m.user = CM.User{
		ID: 2, Username: create.Username, Inbound: create.Inbound, Type: create.Type,
		SquadIDs: create.SquadIDs, UUID: create.UUID, Password: create.Password,
		Secret: create.Secret, AuthorizedKeys: create.AuthorizedKeys, Flow: create.Flow, AlterID: create.AlterID,
	}
	return m.user, nil
}

func TestFleetProjectionChangesUserProtocol(t *testing.T) {
	manager := &fleetUserTypeManager{user: CM.User{
		ID: 1, Username: "alice", Inbound: "proxy-in", Type: "vless", SquadIDs: []int{7},
		UUID: "11111111-1111-4111-8111-111111111111",
	}}
	wanted := CM.UserCreate{
		Username: "alice", Inbound: "proxy-in", Type: "vmess", SquadIDs: []int{7},
		UUID: "22222222-2222-4222-8222-222222222222",
	}
	if err := syncFleetUsers(manager, map[int]struct{}{7: {}}, map[string]CM.UserCreate{userKey("alice", "proxy-in"): wanted}); err != nil {
		t.Fatal(err)
	}
	if manager.user.Type != wanted.Type || manager.user.UUID != wanted.UUID {
		t.Fatalf("protocol change was not applied: type=%s uuid=%s", manager.user.Type, manager.user.UUID)
	}
}
