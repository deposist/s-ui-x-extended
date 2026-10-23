package service

import (
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Client limits are per client, never shared between them: each client gets its
// own speed cap and its own connection cap, and every server enforces the same
// numbers for that client.

// GetClientLimits reads one client's limits. A client with no saved limits has
// none of them, which is the pre-fleet behaviour: unlimited.
func (s *NodeService) GetClientLimits(clientID uint) (*model.ClientLimits, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	limits := &model.ClientLimits{ClientId: clientID, SpeedShared: true}
	err := db.Model(model.ClientLimits{}).Where("client_id = ?", clientID).First(limits).Error
	if err != nil && !database.IsNotFound(err) {
		return nil, err
	}
	return limits, nil
}

// SetClientLimits stores one client's limits, replacing whatever was there.
func (s *NodeService) SetClientLimits(clientID uint, limits model.ClientLimits) (*model.ClientLimits, error) {
	db := s.db()
	if db == nil {
		return nil, common.NewError("database is not initialized")
	}
	if clientID == 0 {
		return nil, common.NewError("client id is required")
	}
	var exists int64
	if err := db.Model(model.Client{}).Where("id = ?", clientID).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, common.NewError("client not found")
	}
	limits.ClientId = clientID
	limits.UpdatedAt = time.Now().Unix()
	if limits.SpeedShared {
		// A shared limit is one number; keeping the split fields would make the
		// stored row contradict the mode the operator chose.
		limits.DownloadSpeed = 0
		limits.UploadSpeed = 0
	} else {
		limits.Speed = 0
	}
	if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&limits).Error; err != nil {
		return nil, err
	}
	return &limits, nil
}

// ClearClientLimits drops one client's limits, restoring unlimited. It runs on
// the caller's transaction so a client delete takes its limits with it.
func ClearClientLimits(tx *gorm.DB, clientID uint) error {
	if tx == nil || clientID == 0 {
		return nil
	}
	return tx.Where("client_id = ?", clientID).Delete(model.ClientLimits{}).Error
}
