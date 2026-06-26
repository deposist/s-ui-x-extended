package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

type ProviderService struct{}

// RefreshProviderHealth aggregates provider health from the latest outbound
// health snapshots and stores it in the bounded in-memory provider snapshot map.
func RefreshProviderHealth(now time.Time) {
	snapshots := (&ProviderService{}).AggregateProviderHealth(now)
	keep := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		keep = append(keep, snapshot.Tag)
		SetProviderHealth(snapshot)
	}
	PruneProviderHealth(keep)
}

// AggregateProviderHealth builds a ProviderHealthSnapshot for every provider
// from the latest outbound health snapshots. It does not probe on the request
// path. Malformed provider members are ignored instead of panicking.
func (p *ProviderService) AggregateProviderHealth(now time.Time) []ProviderHealthSnapshot {
	db := database.GetDB()
	if db == nil {
		return nil
	}
	var providers []*model.Provider
	if err := db.Model(model.Provider{}).Scan(&providers).Error; err != nil {
		return nil
	}
	snapshots := make([]ProviderHealthSnapshot, 0, len(providers))
	for _, prov := range providers {
		snapshot := ProviderHealthSnapshot{
			Tag:       prov.Tag,
			UpdatedAt: now.Unix(),
			Status:    "unknown",
		}
		checkedMembers := 0
		if prov.Type == "inline" {
			opts := optionsMapOf(prov.Options)
			if outs, _ := opts["outbounds"].([]any); len(outs) > 0 {
				for _, item := range outs {
					member, ok := item.(map[string]any)
					if !ok {
						continue
					}
					tag, _ := member["tag"].(string)
					if tag == "" {
						continue
					}
					snapshot.OutboundCount++
					if h, ok := OutboundHealthSnapshotFor(tag); ok {
						checkedMembers++
						if len(snapshot.Members) < providerHealthMaxMembers {
							snapshot.Members = append(snapshot.Members, h)
						}
						if h.Status == "healthy" || h.Status == "unknown" {
							snapshot.HealthyCount++
						}
					}
				}
			}
		}
		checked := checkedMembers
		if snapshot.OutboundCount > 0 {
			switch {
			case checked == 0:
				snapshot.Status = "unknown"
				snapshot.LastError = "no recent member health snapshots"
			case snapshot.HealthyCount == snapshot.OutboundCount:
				snapshot.Status = "healthy"
			case snapshot.HealthyCount == 0 && checked == snapshot.OutboundCount:
				snapshot.Status = "down"
				snapshot.LastError = fmt.Sprintf("%d members failed health check", checked)
			default:
				snapshot.Status = "degraded"
				failed := checked - snapshot.HealthyCount
				missing := snapshot.OutboundCount - checked
				if failed > 0 || missing > 0 {
					snapshot.LastError = fmt.Sprintf("%d failed, %d without recent snapshot", failed, missing)
				}
			}
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func (p *ProviderService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	providers := []*model.Provider{}
	err := db.Model(model.Provider{}).Scan(&providers).Error
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	for _, provider := range providers {
		provData := map[string]interface{}{
			"id":   provider.Id,
			"type": provider.Type,
			"tag":  provider.Tag,
		}
		if provider.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(provider.Options, &restFields); err != nil {
				return nil, err
			}
			for k, v := range restFields {
				provData[k] = v
			}
		}
		data = append(data, provData)
	}
	return &data, nil
}

func (p *ProviderService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var providersJson []json.RawMessage
	var providers []*model.Provider
	err := db.Model(model.Provider{}).Scan(&providers).Error
	if err != nil {
		return nil, err
	}
	for _, provider := range providers {
		providerJson, err := provider.MarshalJSON()
		if err != nil {
			return nil, err
		}
		providersJson = append(providersJson, providerJson)
	}
	return providersJson, nil
}

func (s *ProviderService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	var err error

	switch act {
	case "new", "edit":
		var provider model.Provider
		err = provider.UnmarshalJSON(data)
		if err != nil {
			return err
		}

		if provider.Tag == "" {
			return common.NewError("provider tag is required")
		}
		if provider.Type == "" {
			return common.NewError("provider type is required")
		}

		// Defensive: an "edit" must update the existing row. If the payload
		// omitted the id, resolve it by the unique tag so Save updates instead
		// of inserting (which would fail the unique-tag constraint).
		if act == "edit" && provider.Id == 0 {
			var id uint
			err = tx.Model(model.Provider{}).Select("id").Where("tag = ?", provider.Tag).Scan(&id).Error
			if err != nil {
				return err
			}
			provider.Id = id
		}

		err = tx.Save(&provider).Error
		if err != nil {
			return err
		}
	case "del":
		var tag string
		err = json.Unmarshal(data, &tag)
		if err != nil {
			return err
		}
		err = tx.Where("tag = ?", tag).Delete(model.Provider{}).Error
		if err != nil {
			return err
		}
	default:
		return common.NewErrorf("unknown action: %s", act)
	}
	return nil
}
