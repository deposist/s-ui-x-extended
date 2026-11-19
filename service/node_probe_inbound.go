package service

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

type peerProbeInboundChange struct {
	tag      string
	previous json.RawMessage
	next     json.RawMessage
}

func probeOnlyInboundIDs(before, after []byte) []uint {
	var previous, next []peerUser
	if json.Unmarshal(before, &previous) != nil || json.Unmarshal(after, &next) != nil {
		return nil
	}
	var ids []uint
	withoutProbe := func(users []peerUser) []peerUser {
		return slices.DeleteFunc(users, func(user peerUser) bool {
			if user.ProbeDestination == "" {
				return false
			}
			ids = append(ids, user.InboundID)
			return true
		})
	}
	if !slices.Equal(withoutProbe(previous), withoutProbe(next)) {
		return nil
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

func peerProbeInboundChanges(tx *gorm.DB, ids []uint, previous, next []byte) ([]peerProbeInboundChange, error) {
	var tags []string
	if err := tx.Model(&model.Inbound{}).Where("id IN ? AND type = ?", ids, "vless").Pluck("tag", &tags).Error; err != nil {
		return nil, err
	}
	oldInbounds, err := peerConfigInbounds(previous)
	if err != nil {
		return nil, err
	}
	newInbounds, err := peerConfigInbounds(next)
	if err != nil {
		return nil, err
	}
	changes := make([]peerProbeInboundChange, 0, len(tags))
	for _, tag := range tags {
		if oldInbounds[tag] == nil || newInbounds[tag] == nil {
			return nil, errors.New("traffic check inbound is unavailable in the local proxy")
		}
		changes = append(changes, peerProbeInboundChange{tag: tag, previous: oldInbounds[tag], next: newInbounds[tag]})
	}
	return changes, nil
}

func peerConfigInbounds(data []byte) (map[string]json.RawMessage, error) {
	var config struct {
		Inbounds []json.RawMessage `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	result := make(map[string]json.RawMessage, len(config.Inbounds))
	for _, raw := range config.Inbounds {
		var meta struct {
			Tag string `json:"tag"`
		}
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, err
		}
		result[meta.Tag] = raw
	}
	return result, nil
}
