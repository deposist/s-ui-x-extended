package service

import (
	"encoding/json"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/util/common"

	"gorm.io/gorm"
)

type ProviderService struct{}

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
