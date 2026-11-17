package service

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/gofrs/uuid/v5"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

type fleetCredential struct {
	uuid           string
	username       string
	password       string
	secret         string
	flow           string
	alterID        int
	authorizedKeys []string
}

type fleetCredentials map[string]fleetCredential

func (c fleetCredentials) forType(inboundType string) (fleetCredential, bool) {
	field, supported := map[string]string{
		"anytls": "anytls", "http": "http", "hysteria": "hysteria", "hysteria2": "hysteria2",
		"mixed": "mixed", "mtproxy": "mtproxy", "naive": "naive", "socks": "socks",
		"ssh": "ssh", "trojan": "trojan", "trusttunnel": "trusttunnel", "tuic": "tuic",
		"vless": "vless", "vmess": "vmess",
	}[inboundType]
	if !supported {
		return fleetCredential{}, false
	}
	credential, ok := c[field]
	if !ok {
		return fleetCredential{}, false
	}
	switch inboundType {
	case "vless":
		if _, err := uuid.FromString(credential.uuid); err != nil {
			return fleetCredential{}, false
		}
	case "vmess":
		if _, err := uuid.FromString(credential.uuid); err != nil {
			return fleetCredential{}, false
		}
	case "trojan", "hysteria", "hysteria2":
		if credential.password == "" {
			return fleetCredential{}, false
		}
	case "tuic":
		if _, err := uuid.FromString(credential.uuid); err != nil || credential.password == "" {
			return fleetCredential{}, false
		}
	case "ssh":
		if credential.password == "" && len(credential.authorizedKeys) == 0 {
			return fleetCredential{}, false
		}
	case "mtproxy":
		if credential.secret == "" {
			return fleetCredential{}, false
		}
	}
	return credential, true
}

func decodeClientCredentials(raw json.RawMessage) (fleetCredentials, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return fleetCredentials{}, nil
	}
	var blocks map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("decode client config: %w", err)
	}
	credentials := make(fleetCredentials, len(blocks))
	for field, block := range blocks {
		var value struct {
			UUID                string   `json:"uuid"`
			Username            string   `json:"username"`
			Password            string   `json:"password"`
			Secret              string   `json:"secret"`
			AuthString          string   `json:"auth_str"`
			Flow                string   `json:"flow"`
			AlterID             int      `json:"alterId"`
			AuthorizedKeys      []string `json:"authorizedKeys"`
			AuthorizedKeysSnake []string `json:"authorized_keys"`
		}
		if err := json.Unmarshal(block, &value); err != nil {
			return nil, fmt.Errorf("decode %s client credentials: %w", field, err)
		}
		if len(value.AuthorizedKeys) == 0 {
			value.AuthorizedKeys = value.AuthorizedKeysSnake
		}
		if field == "hysteria" {
			value.Password = value.AuthString
		}
		credentials[field] = fleetCredential{uuid: value.UUID, username: value.Username, password: value.Password, secret: value.Secret, flow: value.Flow, alterID: value.AlterID, authorizedKeys: value.AuthorizedKeys}
	}
	return credentials, nil
}

func sortedInts(values []int) []int {
	result := append([]int(nil), values...)
	sort.Ints(result)
	return result
}

func mergeInts(a []int, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	merged := make([]int, 0, len(a)+len(b))
	for _, id := range a {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}
	for _, id := range b {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}
	return sortedInts(merged)
}

func withoutManagedSquads(squadIDs []int, managed map[int]struct{}) []int {
	remaining := make([]int, 0, len(squadIDs))
	for _, id := range squadIDs {
		if _, ok := managed[id]; !ok {
			remaining = append(remaining, id)
		}
	}
	return sortedInts(remaining)
}

func managedSquad(squadIDs []int, managed map[int]struct{}) bool {
	for _, id := range squadIDs {
		if _, ok := managed[id]; ok {
			return true
		}
	}
	return false
}

func credentialsDiffer(current CM.User, wanted CM.UserCreate) bool {
	return current.Type != wanted.Type || current.UUID != wanted.UUID || current.Password != wanted.Password || current.Secret != wanted.Secret || current.Flow != wanted.Flow || current.AlterID != wanted.AlterID || !slices.Equal(current.AuthorizedKeys, wanted.AuthorizedKeys)
}

func userCreateFrom(current CM.User, squadIDs []int) CM.UserCreate {
	return CM.UserCreate{SquadIDs: sortedInts(squadIDs), Username: current.Username, Inbound: current.Inbound, Type: current.Type, UUID: current.UUID, Password: current.Password, Secret: current.Secret, AuthorizedKeys: current.AuthorizedKeys, Flow: current.Flow, AlterID: current.AlterID}
}

func bandwidthCreateFrom(current CM.BandwidthLimiter, squadIDs []int) CM.BandwidthLimiterCreate {
	return CM.BandwidthLimiterCreate{SquadIDs: sortedInts(squadIDs), Username: current.Username, Outbound: current.Outbound, Strategy: current.Strategy, ConnectionType: current.ConnectionType, Mode: current.Mode, FlowKeys: current.FlowKeys, Speed: current.Speed}
}

func connectionCreateFrom(current CM.ConnectionLimiter, squadIDs []int) CM.ConnectionLimiterCreate {
	return CM.ConnectionLimiterCreate{SquadIDs: sortedInts(squadIDs), Username: current.Username, Outbound: current.Outbound, Strategy: current.Strategy, ConnectionType: current.ConnectionType, LockType: current.LockType, Count: current.Count}
}

func syncFleetUsers(manager CM.Manager, managed map[int]struct{}, desired map[string]CM.UserCreate) error {
	existing, err := manager.GetUsers(nil)
	if err != nil {
		return err
	}
	byKey := make(map[string]CM.User, len(existing))
	for _, u := range existing {
		byKey[userKey(u.Username, u.Inbound)] = u
	}
	keys := make([]string, 0, len(desired))
	for key := range desired {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		wanted := desired[key]
		wanted.SquadIDs = sortedInts(wanted.SquadIDs)
		current, ok := byKey[key]
		if !ok {
			if _, err := manager.CreateUser(wanted); err != nil {
				return fmt.Errorf("create fleet user %q: %w", wanted.Username, err)
			}
			continue
		}
		delete(byKey, key)
		if !managedSquad(current.SquadIDs, managed) {
			return fmt.Errorf("fleet user key %q conflicts with an unmanaged manager record", key)
		}
		foreign := withoutManagedSquads(current.SquadIDs, managed)
		wanted.SquadIDs = mergeInts(wanted.SquadIDs, foreign)
		changedSquads := !sameSquadSet(current.SquadIDs, wanted.SquadIDs)
		changedCredentials := credentialsDiffer(current, wanted)
		if !changedSquads && !changedCredentials {
			continue
		}
		if changedCredentials && len(foreign) > 0 {
			return fmt.Errorf("fleet user key %q also belongs to unmanaged squads; refusing credential change", key)
		}
		if changedSquads || current.Type != wanted.Type {
			if _, err := manager.DeleteUser(current.ID); err != nil {
				return err
			}
			if _, err := manager.CreateUser(wanted); err != nil {
				return err
			}
			continue
		}
		if _, err := manager.UpdateUser(current.ID, CM.UserUpdate{UUID: wanted.UUID, Password: wanted.Password, Secret: wanted.Secret, AuthorizedKeys: wanted.AuthorizedKeys, Flow: wanted.Flow, AlterID: wanted.AlterID}); err != nil {
			return err
		}
	}
	for _, u := range byKey {
		if !managedSquad(u.SquadIDs, managed) {
			continue
		}
		remaining := withoutManagedSquads(u.SquadIDs, managed)
		if _, err := manager.DeleteUser(u.ID); err != nil {
			return err
		}
		if len(remaining) > 0 {
			if _, err := manager.CreateUser(userCreateFrom(u, remaining)); err != nil {
				return err
			}
		}
	}
	return nil
}

func sameBandwidthSettings(c CM.BandwidthLimiter, w CM.BandwidthLimiterCreate) bool {
	return c.Strategy == w.Strategy && c.ConnectionType == w.ConnectionType && c.Mode == w.Mode && c.RawSpeed == rawSpeed(w.Speed) && slices.Equal(c.FlowKeys, w.FlowKeys)
}
func sameConnectionSettings(c CM.ConnectionLimiter, w CM.ConnectionLimiterCreate) bool {
	return c.Strategy == w.Strategy && c.ConnectionType == w.ConnectionType && c.LockType == w.LockType && c.Count == w.Count
}

func mergeBandwidthDesired(desired map[string]CM.BandwidthLimiterCreate, key string, v CM.BandwidthLimiterCreate) error {
	if old, ok := desired[key]; ok {
		if old.Strategy != v.Strategy || old.ConnectionType != v.ConnectionType || old.Mode != v.Mode || old.Speed != v.Speed {
			return fmt.Errorf("conflicting bandwidth limiter policy for client %q outbound %q", v.Username, v.Outbound)
		}
		old.SquadIDs = mergeInts(old.SquadIDs, v.SquadIDs)
		desired[key] = old
		return nil
	}
	v.SquadIDs = sortedInts(v.SquadIDs)
	desired[key] = v
	return nil
}
func mergeConnectionDesired(desired map[string]CM.ConnectionLimiterCreate, key string, v CM.ConnectionLimiterCreate) error {
	if old, ok := desired[key]; ok {
		if old.Strategy != v.Strategy || old.ConnectionType != v.ConnectionType || old.LockType != v.LockType || old.Count != v.Count {
			return fmt.Errorf("conflicting connection limiter policy for client %q outbound %q", v.Username, v.Outbound)
		}
		old.SquadIDs = mergeInts(old.SquadIDs, v.SquadIDs)
		desired[key] = old
		return nil
	}
	v.SquadIDs = sortedInts(v.SquadIDs)
	desired[key] = v
	return nil
}

func syncFleetLimiters(manager CM.Manager, managed map[int]struct{}, desiredBandwidth map[string]CM.BandwidthLimiterCreate, desiredConnections map[string]CM.ConnectionLimiterCreate) error {
	bandwidth, err := manager.GetBandwidthLimiters(nil)
	if err != nil {
		return err
	}
	knownB := make(map[string]CM.BandwidthLimiter, len(bandwidth))
	for _, v := range bandwidth {
		knownB[limiterKey(v.Username, v.Outbound)] = v
	}
	keys := make([]string, 0, len(desiredBandwidth))
	for k := range desiredBandwidth {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		w := desiredBandwidth[key]
		c, ok := knownB[key]
		if !ok {
			if _, err := manager.CreateBandwidthLimiter(w); err != nil {
				return err
			}
			continue
		}
		delete(knownB, key)
		if !managedSquad(c.SquadIDs, managed) {
			return fmt.Errorf("bandwidth limiter key %q conflicts with an unmanaged manager record", key)
		}
		foreign := withoutManagedSquads(c.SquadIDs, managed)
		w.SquadIDs = mergeInts(w.SquadIDs, foreign)
		sc := !sameSquadSet(c.SquadIDs, w.SquadIDs)
		pc := !sameBandwidthSettings(c, w)
		if !sc && !pc {
			continue
		}
		if pc && len(foreign) > 0 {
			return fmt.Errorf("bandwidth limiter key %q also belongs to unmanaged squads", key)
		}
		if sc {
			if _, err := manager.DeleteBandwidthLimiter(c.ID); err != nil {
				return err
			}
			if _, err := manager.CreateBandwidthLimiter(w); err != nil {
				return err
			}
			continue
		}
		if _, err := manager.UpdateBandwidthLimiter(c.ID, CM.BandwidthLimiterUpdate{Username: w.Username, Outbound: w.Outbound, Strategy: w.Strategy, ConnectionType: w.ConnectionType, Mode: w.Mode, FlowKeys: w.FlowKeys, Speed: w.Speed}); err != nil {
			return err
		}
	}
	for _, v := range knownB {
		if !managedSquad(v.SquadIDs, managed) {
			continue
		}
		remaining := withoutManagedSquads(v.SquadIDs, managed)
		if _, err := manager.DeleteBandwidthLimiter(v.ID); err != nil {
			return err
		}
		if len(remaining) > 0 {
			if _, err := manager.CreateBandwidthLimiter(bandwidthCreateFrom(v, remaining)); err != nil {
				return err
			}
		}
	}
	connections, err := manager.GetConnectionLimiters(nil)
	if err != nil {
		return err
	}
	knownC := make(map[string]CM.ConnectionLimiter, len(connections))
	for _, v := range connections {
		knownC[limiterKey(v.Username, v.Outbound)] = v
	}
	keys = keys[:0]
	for k := range desiredConnections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		w := desiredConnections[key]
		c, ok := knownC[key]
		if !ok {
			if _, err := manager.CreateConnectionLimiter(w); err != nil {
				return err
			}
			continue
		}
		delete(knownC, key)
		if !managedSquad(c.SquadIDs, managed) {
			return fmt.Errorf("connection limiter key %q conflicts with an unmanaged manager record", key)
		}
		foreign := withoutManagedSquads(c.SquadIDs, managed)
		w.SquadIDs = mergeInts(w.SquadIDs, foreign)
		sc := !sameSquadSet(c.SquadIDs, w.SquadIDs)
		pc := !sameConnectionSettings(c, w)
		if !sc && !pc {
			continue
		}
		if pc && len(foreign) > 0 {
			return fmt.Errorf("connection limiter key %q also belongs to unmanaged squads", key)
		}
		if sc {
			if _, err := manager.DeleteConnectionLimiter(c.ID); err != nil {
				return err
			}
			if _, err := manager.CreateConnectionLimiter(w); err != nil {
				return err
			}
			continue
		}
		if _, err := manager.UpdateConnectionLimiter(c.ID, CM.ConnectionLimiterUpdate{Username: w.Username, Outbound: w.Outbound, Strategy: w.Strategy, ConnectionType: w.ConnectionType, LockType: w.LockType, Count: w.Count}); err != nil {
			return err
		}
	}
	for _, v := range knownC {
		if !managedSquad(v.SquadIDs, managed) {
			continue
		}
		remaining := withoutManagedSquads(v.SquadIDs, managed)
		if _, err := manager.DeleteConnectionLimiter(v.ID); err != nil {
			return err
		}
		if len(remaining) > 0 {
			if _, err := manager.CreateConnectionLimiter(connectionCreateFrom(v, remaining)); err != nil {
				return err
			}
		}
	}
	return nil
}
