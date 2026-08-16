package service

import (
	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)

// AWGStatusInfo is the admin-facing managed-AWG health snapshot. It carries no
// key material: encryption availability is a boolean.
type AWGStatusInfo struct {
	ManagedEndpoints       int64 `json:"managedEndpoints"`
	CoreReachable          bool  `json:"coreReachable"`
	EncryptionKeyAvailable bool  `json:"encryptionKeyAvailable"`
	Desired                int64 `json:"desired"`
	Provisioned            int64 `json:"provisioned"`
	Pending                int64 `json:"pending"`
	Errors                 int64 `json:"errors"`
}

// CollectAWGStatus aggregates the managed-AWG state used by the settings page
// status line: how many endpoints are managed, device counters, core
// reachability, and whether the device-key encryption key is usable.
func CollectAWGStatus() (AWGStatusInfo, error) {
	endpointIDs, err := ListManagedAWGEndpoints(database.GetDB())
	if err != nil {
		return AWGStatusInfo{}, err
	}
	info := AWGStatusInfo{ManagedEndpoints: int64(len(endpointIDs))}
	db := database.GetDB()
	if err := db.Model(&model.AWGDevice{}).Where("desired_enabled = ?", true).Count(&info.Desired).Error; err != nil {
		return AWGStatusInfo{}, err
	}
	_ = db.Model(&model.AWGDevice{}).Where("provisioned = ?", true).Count(&info.Provisioned).Error
	_ = db.Model(&model.AWGDevice{}).Where("sync_state <> ?", "in_sync").Count(&info.Pending).Error
	_ = db.Model(&model.AWGDevice{}).Where("last_error <> ''").Count(&info.Errors).Error
	if core := DefaultRuntime().Core(); core != nil {
		info.CoreReachable = core.IsRunning()
	}
	_, encryptionErr := NewAWGCipherFromEnv()
	info.EncryptionKeyAvailable = encryptionErr == nil
	return info, nil
}
