package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	CM "github.com/sagernet/sing-box/service/manager/constant"
)

func userKey(username, inbound string) string     { return username + "\x00" + inbound }
func limiterKey(username, outbound string) string { return username + "\x00" + outbound }

func sameUserCredentials(a, b CM.UserCreate) bool {
	return a.Type == b.Type && a.UUID == b.UUID && a.Password == b.Password && a.Secret == b.Secret && a.Flow == b.Flow && a.AlterID == b.AlterID && strings.Join(a.AuthorizedKeys, "\x00") == strings.Join(b.AuthorizedKeys, "\x00")
}

func (s *fleetState) projectClient(client model.Client, users map[string]CM.UserCreate, bandwidth map[string]CM.BandwidthLimiterCreate, connections map[string]CM.ConnectionLimiterCreate) error {
	if strings.TrimSpace(client.Name) == "" {
		return nil
	}
	credentials, err := decodeClientCredentials(client.Config)
	if err != nil {
		logger.Warning("client ", client.Name, " credentials unreadable, not projected: ", err)
		return nil
	}
	for _, node := range s.nodes {
		if !node.row.Enabled || !s.assigned(client.Id, node.row.UUID) {
			continue
		}
		projectedUsernames := make(map[string]struct{})
		for tag, type_ := range node.tags {
			credential, ok := credentials.forType(type_)
			if !ok {
				continue
			}
			username := client.Name
			if credential.username != "" {
				username = credential.username
			}
			key := userKey(username, tag)
			wanted := CM.UserCreate{Username: username, Inbound: tag, Type: type_, UUID: credential.uuid, Password: credential.password, Secret: credential.secret, Flow: credential.flow, AlterID: credential.alterID, AuthorizedKeys: credential.authorizedKeys, SquadIDs: []int{node.squadID}}
			if current, exists := users[key]; exists {
				if !sameUserCredentials(current, wanted) {
					return fmt.Errorf("clients sharing username %q have conflicting credentials for inbound %q", username, tag)
				}
				current.SquadIDs = append(current.SquadIDs, node.squadID)
				users[key] = current
			} else {
				users[key] = wanted
			}
			projectedUsernames[username] = struct{}{}
		}
		if limits, ok := s.limits[client.Id]; ok {
			if len(projectedUsernames) == 0 {
				projectedUsernames[client.Name] = struct{}{}
			}
			for username := range projectedUsernames {
				if err := s.projectLimits(username, node, limits, bandwidth, connections); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *fleetState) projectLimits(username string, node fleetNode, limits model.ClientLimits, bandwidth map[string]CM.BandwidthLimiterCreate, connections map[string]CM.ConnectionLimiterCreate) error {
	addBandwidth := func(outbound, mode string, speed uint64) error {
		if outbound == "" || speed == 0 {
			return nil
		}
		value := CM.BandwidthLimiterCreate{SquadIDs: []int{node.squadID}, Username: username, Outbound: outbound, Strategy: "global", Mode: mode, Speed: strconv.FormatUint(speed, 10) + "Bps"}
		return mergeBandwidthDesired(bandwidth, limiterKey(username, outbound), value)
	}
	if limits.SpeedShared {
		if err := addBandwidth(node.roles.SpeedShared, "bidirectional", limits.Speed); err != nil {
			return err
		}
	} else {
		if err := addBandwidth(node.roles.SpeedDown, "download", limits.DownloadSpeed); err != nil {
			return err
		}
		if err := addBandwidth(node.roles.SpeedUp, "upload", limits.UploadSpeed); err != nil {
			return err
		}
	}
	if node.roles.Connections != "" && limits.Connections > 0 {
		value := CM.ConnectionLimiterCreate{SquadIDs: []int{node.squadID}, Username: username, Outbound: node.roles.Connections, Strategy: "connection", LockType: "manager", Count: limits.Connections}
		return mergeConnectionDesired(connections, limiterKey(username, node.roles.Connections), value)
	}
	return nil
}

func (s *fleetState) assigned(clientID uint, uuid string) bool {
	for _, assignedUUID := range s.assign[clientID] {
		if assignedUUID == uuid {
			return true
		}
	}
	return false
}

func rawSpeed(speed string) uint64 {
	trimmed := strings.TrimSuffix(strings.TrimSpace(speed), "Bps")
	value, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0
	}
	return value
}
