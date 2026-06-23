package service

import (
	"encoding/json"
	"os"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

type ServicesService struct {
	Runtime *Runtime
}

func (s *ServicesService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (s *ServicesService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	services := []model.Service{}
	err := db.Model(model.Service{}).Scan(&services).Error
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	for _, srv := range services {
		srvData := map[string]interface{}{
			"id":     srv.Id,
			"type":   srv.Type,
			"tag":    srv.Tag,
			"tls_id": srv.TlsId,
		}
		if srv.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(srv.Options, &restFields); err != nil {
				return nil, err
			}
			for k, v := range restFields {
				srvData[k] = v
			}
		}

		data = append(data, srvData)
	}
	return &data, nil
}

func (s *ServicesService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var servicesJson []json.RawMessage
	var services []*model.Service
	err := db.Model(model.Service{}).Preload("Tls").Find(&services).Error
	if err != nil {
		return nil, err
	}
	for _, srv := range services {
		srvJson, err := srv.MarshalJSON()
		if err != nil {
			return nil, err
		}
		servicesJson = append(servicesJson, srvJson)
	}
	return servicesJson, nil
}

func (s *ServicesService) Save(tx *gorm.DB, act string, data json.RawMessage) (*entityCoreChange, error) {
	switch act {
	case "new", "edit":
		var srv model.Service
		if err := srv.UnmarshalJSON(data); err != nil {
			return nil, err
		}

		if srv.TlsId > 0 {
			if err := tx.Model(model.Tls{}).Where("id = ?", srv.TlsId).Find(&srv.Tls).Error; err != nil {
				return nil, err
			}
		}

		var oldTag string
		if srv.Id > 0 {
			if err := tx.Model(model.Service{}).Select("tag").Where("id = ?", srv.Id).Find(&oldTag).Error; err != nil {
				return nil, err
			}
		}

		if err := tx.Save(&srv).Error; err != nil {
			return nil, err
		}
		change := &entityCoreChange{reloadIds: []uint{srv.Id}}
		if oldTag != "" && oldTag != srv.Tag {
			change.removeTags = []string{oldTag}
		}
		return change, nil
	case "del":
		var tag string
		if err := json.Unmarshal(data, &tag); err != nil {
			return nil, err
		}
		if err := tx.Where("tag = ?", tag).Delete(model.Service{}).Error; err != nil {
			return nil, err
		}
		return &entityCoreChange{removeTags: []string{tag}}, nil
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}
}

// RemoveServicesFromCore removes the given service tags from the running core.
// Missing tags are tolerated so removals stay idempotent; with no running core
// there is nothing to remove.
func (s *ServicesService) RemoveServicesFromCore(tags []string) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	for _, tag := range tags {
		if err := coreInstance.RemoveService(tag); err != nil && err != os.ErrInvalid {
			return err
		}
	}
	return nil
}

func (s *ServicesService) RestartServices(tx *gorm.DB, ids []uint) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	var services []*model.Service
	err := tx.Model(model.Service{}).Preload("Tls").Where("id in ?", ids).Find(&services).Error
	if err != nil {
		return err
	}
	for _, srv := range services {
		err = coreInstance.RemoveService(srv.Tag)
		if err != nil && err != os.ErrInvalid {
			return err
		}
		srvConfig, err := srv.MarshalJSON()
		if err != nil {
			return err
		}
		err = coreInstance.AddService(srvConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
