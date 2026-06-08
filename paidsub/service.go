package paidsub

import (
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/sub"

	"gorm.io/gorm"
)

// PaidSubService is the narrow, least-privilege facade the bot and HTTP
// handlers use. It deliberately exposes ONLY client-self operations (resolve by
// Telegram id, read links/QR/stats, auto-register, renew the resolved client)
// and never settings/users/tokens/inbound management. All client mutations are
// scoped to the single client bound to the acting Telegram user.
type PaidSubService struct {
	Setting service.SettingService
	Client  service.ClientService
	Stats   service.StatsService
	Link    sub.LinkService
}

// NewService builds the facade with default-runtime-backed core services.
func NewService() *PaidSubService {
	return &PaidSubService{}
}

func nowUnix() int64 { return time.Now().Unix() }

// ClientByTgUserId resolves the single client bound to a Telegram user via the
// paidsub_bindings table. It returns gorm.ErrRecordNotFound when the user is
// not bound. Both enabled and disabled clients are returned (a disabled/expired
// client still needs to see status and buy a renewal). tgUserId must be > 0.
func (s *PaidSubService) ClientByTgUserId(tgUserId int64) (*model.Client, error) {
	if tgUserId <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	db := database.GetDB()
	var client model.Client
	err := db.Model(&model.Client{}).
		Joins("JOIN paidsub_bindings b ON b.client_id = clients.id").
		Where("b.tg_user_id = ?", tgUserId).
		First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// BindingForClient returns the binding row for a client, or ErrRecordNotFound.
func (s *PaidSubService) BindingForClient(clientId uint) (*Binding, error) {
	db := database.GetDB()
	var b Binding
	if err := db.Where("client_id = ?", clientId).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

// SetBinding maps tgUserId to clientId (one-to-one, latest wins). It releases
// any existing binding that holds either side, then upserts, all in one
// transaction so the UNIQUE constraints can never be violated mid-operation.
func (s *PaidSubService) SetBinding(clientId uint, tgUserId int64) error {
	if clientId == 0 || tgUserId <= 0 {
		return gorm.ErrInvalidData
	}
	db := database.GetDB()
	now := nowUnix()
	return db.Transaction(func(tx *gorm.DB) error {
		// Release any row that currently holds this tg id or this client id.
		if err := tx.Where("tg_user_id = ? OR client_id = ?", tgUserId, clientId).
			Delete(&Binding{}).Error; err != nil {
			return err
		}
		return tx.Create(&Binding{
			ClientId:  clientId,
			TgUserId:  tgUserId,
			CreatedAt: now,
			UpdatedAt: now,
		}).Error
	})
}

// UnbindClient removes the binding for a client (no-op if absent).
func (s *PaidSubService) UnbindClient(clientId uint) error {
	db := database.GetDB()
	return db.Where("client_id = ?", clientId).Delete(&Binding{}).Error
}
