package service

import (
	"encoding/json"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

const CoreFailoverType = "core-failover"

func stringListOption(options json.RawMessage, key string) ([]string, error) {
	opts := optionsMapOf(options)
	items, _ := opts[key].([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		tag, ok := item.(string)
		if !ok || tag == "" {
			return nil, common.NewErrorf("%s member must be a string tag", key)
		}
		out = append(out, tag)
	}
	return out, nil
}

func outboundCoreJSON(db *gorm.DB, row model.Outbound) (json.RawMessage, error) {
	if row.Type == FailoverType {
		return assembleFailoverForCore(row, DirectFallbackTag(db))
	}
	if row.Type == CoreFailoverType {
		return assembleCoreFailoverOutboundForCore(db, row)
	}
	return row.MarshalJSON()
}

func assembleCoreFailoverOutboundForCore(db *gorm.DB, row model.Outbound) (json.RawMessage, error) {
	tags, err := stringListOption(row.Options, "outbounds")
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, common.NewErrorf("core-failover outbound %q has no members", row.Tag)
	}
	opts := optionsMapOf(row.Options)
	members := make([]json.RawMessage, 0, len(tags))
	for _, tag := range tags {
		var member model.Outbound
		if err := db.Model(model.Outbound{}).Where("tag = ?", tag).First(&member).Error; err != nil {
			return nil, common.NewErrorf("core-failover member %q does not exist", tag)
		}
		memberJSON, err := outboundCoreJSON(db, member)
		if err != nil {
			return nil, err
		}
		members = append(members, memberJSON)
	}
	core := map[string]any{
		"type":      "failover",
		"tag":       row.Tag,
		"outbounds": members,
	}
	if strategy, _ := opts["strategy"].(string); strategy != "" {
		core["strategy"] = strategy
	}
	if delay, _ := opts["delay"].(string); delay != "" {
		core["delay"] = delay
	}
	return json.Marshal(core)
}

func inboundCoreJSON(s *InboundService, db *gorm.DB, row model.Inbound) (json.RawMessage, error) {
	if row.Type == "bond" || row.Type == CoreFailoverType {
		return s.assembleNestedInboundGroupForCore(db, row)
	}
	inboundJSON, err := row.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if row.Type == "sudoku" {
		var core map[string]any
		if err := json.Unmarshal(inboundJSON, &core); err != nil {
			return nil, err
		}
		delete(core, "master_key")
		inboundJSON, err = json.Marshal(core)
		if err != nil {
			return nil, err
		}
	}
	return s.addUsers(db, inboundJSON, row.Id, row.Type)
}

func (s *InboundService) assembleNestedInboundGroupForCore(db *gorm.DB, row model.Inbound) (json.RawMessage, error) {
	tags, err := stringListOption(row.Options, "inbounds")
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, common.NewErrorf("%s inbound %q has no members", row.Type, row.Tag)
	}
	members := make([]json.RawMessage, 0, len(tags))
	for _, tag := range tags {
		var member model.Inbound
		if err := db.Model(model.Inbound{}).Preload("Tls").Where("tag = ?", tag).First(&member).Error; err != nil {
			return nil, common.NewErrorf("%s inbound member %q does not exist", row.Type, tag)
		}
		memberJSON, err := inboundCoreJSON(s, db, member)
		if err != nil {
			return nil, err
		}
		members = append(members, memberJSON)
	}
	coreType := row.Type
	if row.Type == CoreFailoverType {
		coreType = "failover"
	}
	core := map[string]any{
		"type":     coreType,
		"tag":      row.Tag,
		"inbounds": members,
	}
	return json.Marshal(core)
}

func nestedInboundMemberTags(rows []*model.Inbound) (map[string]bool, error) {
	nested := map[string]bool{}
	for _, row := range rows {
		if row.Type != "bond" && row.Type != CoreFailoverType {
			continue
		}
		tags, err := stringListOption(row.Options, "inbounds")
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			nested[tag] = true
		}
	}
	return nested, nil
}
