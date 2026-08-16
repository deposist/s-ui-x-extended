package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
)

type EndpointService struct {
	WarpService
	Runtime *Runtime
}

func (s *EndpointService) runtime() *Runtime {
	if s != nil {
		return runtimeOrDefault(s.Runtime)
	}
	return DefaultRuntime()
}

func (o *EndpointService) GetAll() (*[]map[string]interface{}, error) {
	db := database.GetDB()
	endpoints := []*model.Endpoint{}
	err := db.Model(model.Endpoint{}).Scan(&endpoints).Error
	if err != nil {
		return nil, err
	}
	var data []map[string]interface{}
	for _, endpoint := range endpoints {
		metadata, metadataErr := parseAWGEndpointMetadata(*endpoint)
		if metadataErr != nil {
			return nil, metadataErr
		}
		epData := map[string]interface{}{
			"id":         endpoint.Id,
			"type":       endpoint.Type,
			"tag":        endpoint.Tag,
			"ext":        endpoint.Ext,
			"awgManaged": metadata.Managed,
		}
		if endpoint.Options != nil {
			var restFields map[string]json.RawMessage
			if err := json.Unmarshal(endpoint.Options, &restFields); err != nil {
				return nil, err
			}
			for k, v := range restFields {
				epData[k] = v
			}
		}
		data = append(data, epData)
	}
	return &data, nil
}

func (o *EndpointService) GetAllConfig(db *gorm.DB) ([]json.RawMessage, error) {
	var endpointsJson []json.RawMessage
	var endpoints []*model.Endpoint
	err := db.Model(model.Endpoint{}).Scan(&endpoints).Error
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints {
		endpointJson, err := endpoint.MarshalJSON()
		if err != nil {
			return nil, err
		}
		endpointsJson = append(endpointsJson, endpointJson)
	}
	return endpointsJson, nil
}

func (s *EndpointService) Save(tx *gorm.DB, act string, data json.RawMessage) (*entityCoreChange, error) {
	switch act {
	case "new", "edit":
		return s.saveEndpointUpsert(tx, act, data)
	case "del":
		return s.saveEndpointDelete(tx, data)
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}
}

func (s *EndpointService) saveEndpointUpsert(tx *gorm.DB, act string, data json.RawMessage) (*entityCoreChange, error) {
	var endpoint model.Endpoint
	if err := endpoint.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	metadata, metadataErr := parseAWGEndpointMetadata(endpoint)
	if metadataErr != nil {
		return nil, metadataErr
	}
	if metadata.Managed {
		if _, err := validateAWGEndpointMetadata(endpoint); err != nil {
			return nil, err
		}
		// Reject core options the device manager cannot work with (a /32
		// address, missing listen_port) at save time instead of failing
		// every background reconcile afterwards.
		if err := validateManagedAWGEndpointOptions(endpoint.Options); err != nil {
			return nil, err
		}
	}
	// Invalid Amnezia obfuscation combinations make the tunnel silently fail
	// to come up (no error on either side), so saving is blocked outright.
	// Existing endpoints are untouched until their next edit.
	if endpoint.Type == "wireguard" || endpoint.Type == "warp" {
		if err := ValidateAmneziaOptions(endpoint.Type, endpoint.Options); err != nil {
			return nil, err
		}
	}
	if act == "edit" && endpoint.Id > 0 {
		var current model.Endpoint
		if err := tx.First(&current, endpoint.Id).Error; err != nil {
			return nil, err
		}
		currentMetadata, err := parseAWGEndpointMetadata(current)
		if err != nil {
			return nil, err
		}
		if currentMetadata.Managed && !awgEndpointPeersEqual(current.Options, endpoint.Options) {
			return nil, fmt.Errorf("managed AWG endpoint peers are controlled by the device manager")
		}
	}

	if endpoint.Type == "warp" {
		if act == "new" {
			if err := s.WarpService.RegisterWarp(&endpoint); err != nil {
				return nil, err
			}
		} else {
			var old_license string
			if err := tx.Model(model.Endpoint{}).Select("json_extract(ext, '$.license_key')").Where("id = ?", endpoint.Id).Find(&old_license).Error; err != nil {
				return nil, err
			}
			if err := s.WarpService.SetWarpLicense(old_license, &endpoint); err != nil {
				return nil, err
			}
		}
	}

	var oldTag string
	if endpoint.Id > 0 {
		if err := tx.Model(model.Endpoint{}).Select("tag").Where("id = ?", endpoint.Id).Find(&oldTag).Error; err != nil {
			return nil, err
		}
	}
	renamed := oldTag != "" && oldTag != endpoint.Tag
	if renamed {
		// Renaming a referenced tag is treated like deleting the old tag: the
		// next core start would fail on the dangling reference.
		refs, err := outboundTagReferences(tx, oldTag, 0, endpoint.Id)
		if err != nil {
			return nil, err
		}
		if len(refs) > 0 {
			return nil, formatTagReferenceError("endpoint", oldTag, refs)
		}
	}

	if err := tx.Save(&endpoint).Error; err != nil {
		return nil, err
	}

	refs, err := outboundTagReferences(tx, endpoint.Tag, 0, endpoint.Id)
	if err != nil {
		return nil, err
	}
	if eager := eagerTagReferences(refs); len(eager) > 0 {
		// Groups, detour dialers and dns/ntp transports capture the adapter at
		// construction; only a full restart re-binds them to the new one.
		return &entityCoreChange{
			needsRestart:  true,
			restartReason: fmt.Sprintf("endpoint %q is captured at construction by %s", endpoint.Tag, eager[0].Locator),
		}, nil
	}
	change := &entityCoreChange{reloadIds: []uint{endpoint.Id}}
	if renamed {
		change.removeTags = []string{oldTag}
	}
	return change, nil
}

func awgEndpointPeersEqual(current, next json.RawMessage) bool {
	var currentOptions, nextOptions map[string]json.RawMessage
	if json.Unmarshal(current, &currentOptions) != nil || json.Unmarshal(next, &nextOptions) != nil {
		return false
	}
	currentPeers := bytes.TrimSpace(currentOptions["peers"])
	nextPeers := bytes.TrimSpace(nextOptions["peers"])
	if len(currentPeers) == 0 {
		currentPeers = []byte("[]")
	}
	if len(nextPeers) == 0 {
		nextPeers = []byte("[]")
	}
	return bytes.Equal(currentPeers, nextPeers)
}

func (s *EndpointService) saveEndpointDelete(tx *gorm.DB, data json.RawMessage) (*entityCoreChange, error) {
	var tag string
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, err
	}
	var ownId uint
	if err := tx.Model(model.Endpoint{}).Select("id").Where("tag = ?", tag).Scan(&ownId).Error; err != nil {
		return nil, err
	}
	if ownId > 0 {
		var accessCount int64
		if err := tx.Model(&model.ClientEndpointAccess{}).Where("endpoint_id = ?", ownId).Count(&accessCount).Error; err != nil {
			return nil, err
		}
		if accessCount > 0 {
			return nil, fmt.Errorf("managed AWG endpoint is assigned to clients")
		}
	}
	refs, err := outboundTagReferences(tx, tag, 0, ownId)
	if err != nil {
		return nil, err
	}
	if len(refs) > 0 {
		return nil, formatTagReferenceError("endpoint", tag, refs)
	}
	if err := tx.Where("tag = ?", tag).Delete(model.Endpoint{}).Error; err != nil {
		return nil, err
	}
	return &entityCoreChange{removeTags: []string{tag}}, nil
}

// RestartEndpoints replaces the given endpoints inside the running core
// (remove by tag, then add the committed definition).
func (s *EndpointService) RestartEndpoints(tx *gorm.DB, ids []uint) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	var endpoints []*model.Endpoint
	if err := tx.Model(model.Endpoint{}).Where("id in ?", ids).Find(&endpoints).Error; err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		if err := coreInstance.RemoveEndpoint(endpoint.Tag); err != nil && err != os.ErrInvalid {
			return err
		}
		endpointConfig, err := endpoint.MarshalJSON()
		if err != nil {
			return err
		}
		if err := coreInstance.AddEndpoint(endpointConfig); err != nil {
			return err
		}
	}
	return nil
}

// RemoveEndpointsFromCore removes the given endpoint tags from the running
// core. Missing tags are tolerated so removals stay idempotent.
func (s *EndpointService) RemoveEndpointsFromCore(tags []string) error {
	coreInstance := s.runtime().Core()
	if coreInstance == nil || !coreInstance.IsRunning() {
		return nil
	}
	for _, tag := range tags {
		if err := coreInstance.RemoveEndpoint(tag); err != nil && err != os.ErrInvalid {
			return err
		}
	}
	return nil
}
