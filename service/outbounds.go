package service

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

// rejectUnknownCallOutboundDialFields blocks call outbounds carrying
// server/server_port before they reach the database. The call outbound schema
// in the core has no dial fields (it joins a room via join_link), so the core
// rejects the saved config with "json: unknown field \"server\"" on every
// start, which crash-loops sing-box until the row is removed by hand.
func rejectUnknownCallOutboundDialFields(data json.RawMessage) error {
	var fields struct {
		Type       string  `json:"type"`
		Server     *string `json:"server"`
		ServerPort *uint16 `json:"server_port"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil // malformed payloads fail later in the model unmarshal
	}
	if fields.Type != "call" || fields.Server == nil && fields.ServerPort == nil {
		return nil
	}
	return fmt.Errorf("call outbounds have no server or port; configure join_link instead (a saved server field makes the core reject the whole config)")
}

type OutboundService struct {
	Runtime *Runtime
}

func (s *OutboundService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (o *OutboundService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	outbounds := []*model.Outbound{}
	err := db.Model(model.Outbound{}).Scan(&outbounds).Error
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	for _, outbound := range outbounds {
		outData := map[string]interface{}{
			"id":   outbound.Id,
			"type": outbound.Type,
			"tag":  outbound.Tag,
		}
		if outbound.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(outbound.Options, &restFields); err != nil {
				return nil, err
			}
			for k, v := range restFields {
				outData[k] = v
			}
		}
		data = append(data, outData)
	}
	return &data, nil
}

func (o *OutboundService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var outboundsJson []json.RawMessage
	var outbounds []*model.Outbound
	err := db.Model(model.Outbound{}).Scan(&outbounds).Error
	if err != nil {
		return nil, err
	}
	for _, outbound := range outbounds {
		outboundJson, err := outboundCoreJSON(db, *outbound)
		if err != nil {
			return nil, err
		}
		outboundsJson = append(outboundsJson, outboundJson)
	}
	return outboundsJson, nil
}

func (s *OutboundService) Save(tx *gorm.DB, act string, data json.RawMessage) (*entityCoreChange, error) {
	switch act {
	case "new", "edit":
		return s.saveOutboundUpsert(tx, data)
	case "del":
		return s.saveOutboundDelete(tx, data)
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}
}

func (s *OutboundService) saveOutboundUpsert(tx *gorm.DB, data json.RawMessage) (*entityCoreChange, error) {
	if err := rejectUnknownCallOutboundDialFields(data); err != nil {
		return nil, err
	}
	var outbound model.Outbound
	if err := outbound.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	if outbound.Type == FailoverType {
		if err := validateFailoverGroup(tx, outbound); err != nil {
			return nil, err
		}
	}
	if outbound.Type == "fallback" {
		if err := validateFallbackGroup(tx, outbound); err != nil {
			return nil, err
		}
	}
	if outbound.Type == "selector" || outbound.Type == "urltest" {
		if err := validateSelectorURLTestGroup(tx, outbound); err != nil {
			return nil, err
		}
	}
	if outbound.Type == "core-failover" {
		if err := validateCoreFailoverOutbound(tx, outbound); err != nil {
			return nil, err
		}
	}
	var oldTag string
	if outbound.Id > 0 {
		if err := tx.Model(model.Outbound{}).Select("tag").Where("id = ?", outbound.Id).Find(&oldTag).Error; err != nil {
			return nil, err
		}
	}
	renamed := oldTag != "" && oldTag != outbound.Tag
	if renamed {
		// Renaming a referenced tag is treated like deleting the old tag: the
		// next core start would fail on the dangling reference.
		refs, err := outboundTagReferences(tx, oldTag, outbound.Id, 0)
		if err != nil {
			return nil, err
		}
		if len(refs) > 0 {
			return nil, formatTagReferenceError("outbound", oldTag, refs)
		}
	}

	if err := tx.Save(&outbound).Error; err != nil {
		return nil, err
	}

	refs, err := outboundTagReferences(tx, outbound.Tag, outbound.Id, 0)
	if err != nil {
		return nil, err
	}
	if eager := eagerTagReferences(refs); len(eager) > 0 {
		// Groups, detour dialers and dns/ntp transports capture the adapter at
		// construction; only a full restart re-binds them to the new one.
		return &entityCoreChange{
			needsRestart:  true,
			restartReason: fmt.Sprintf("outbound %q is captured at construction by %s", outbound.Tag, eager[0].Locator),
		}, nil
	}
	change := &entityCoreChange{reloadIds: []uint{outbound.Id}}
	if renamed {
		change.removeTags = []string{oldTag}
	}
	return change, nil
}

func (s *OutboundService) saveOutboundDelete(tx *gorm.DB, data json.RawMessage) (*entityCoreChange, error) {
	var tag string
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, err
	}
	var ownId uint
	if err := tx.Model(model.Outbound{}).Select("id").Where("tag = ?", tag).Scan(&ownId).Error; err != nil {
		return nil, err
	}
	refs, err := outboundTagReferences(tx, tag, ownId, 0)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		return nil, formatTagReferenceError("outbound", tag, refs)
	}
	if err := tx.Where("tag = ?", tag).Delete(model.Outbound{}).Error; err != nil {
		return nil, err
	}
	return &entityCoreChange{removeTags: []string{tag}}, nil
}

// RestartOutbounds replaces the given outbounds inside the running core
// (remove by tag, then add the committed definition).
func (s *OutboundService) RestartOutbounds(tx *gorm.DB, ids []uint) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	var outbounds []*model.Outbound
	if err := tx.Model(model.Outbound{}).Where("id in ?", ids).Find(&outbounds).Error; err != nil {
		return err
	}
	directTag := DirectFallbackTag(tx)
	for _, outbound := range outbounds {
		if err := coreInstance.RemoveOutbound(outbound.Tag); err != nil && err != os.ErrInvalid {
			return err
		}
		var outboundConfig json.RawMessage
		var err error
		if outbound.Type == FailoverType {
			outboundConfig, err = assembleFailoverForCore(*outbound, directTag)
		} else {
			outboundConfig, err = outbound.MarshalJSON()
		}
		if err != nil {
			return err
		}
		if err := coreInstance.AddOutbound(outboundConfig); err != nil {
			return err
		}
	}
	return nil
}

// RemoveOutboundsFromCore removes the given outbound tags from the running
// core. Missing tags are tolerated so removals stay idempotent.
func (s *OutboundService) RemoveOutboundsFromCore(tags []string) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	for _, tag := range tags {
		if err := coreInstance.RemoveOutbound(tag); err != nil && err != os.ErrInvalid {
			return err
		}
	}
	return nil
}
