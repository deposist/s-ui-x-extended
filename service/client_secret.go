package service

import (
	"strconv"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func (s *ClientService) prepareClientSubSecret(tx *gorm.DB, client *model.Client, preserveExisting bool) error {
	if client.IPLimitMode == "" {
		client.IPLimitMode = "monitor"
	}
	if client.SubSecret != "" {
		return nil
	}
	secret, err := uuid.NewV4()
	if err != nil {
		return err
	}
	candidate := secret.String()
	if preserveExisting && client.Id > 0 {
		result := tx.Model(model.Client{}).
			Where("id = ? AND (sub_secret = ? OR sub_secret IS NULL)", client.Id, "").
			Update("sub_secret", candidate)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			client.SubSecret = candidate
			return nil
		}
		var old model.Client
		if err := tx.Model(model.Client{}).Select("sub_secret").Where("id = ?", client.Id).First(&old).Error; err != nil {
			return err
		}
		if old.SubSecret == "" {
			return gorm.ErrRecordNotFound
		}
		client.SubSecret = old.SubSecret
		return nil
	}
	client.SubSecret = candidate
	return nil
}

func (s *ClientService) RotateSubSecret(id string) (string, error) {
	clientID, err := strconv.ParseUint(id, 10, 64)
	if err != nil || clientID == 0 {
		return "", common.NewError("invalid client id")
	}
	db := database.GetDB()
	var client model.Client
	if err := db.Model(model.Client{}).Select("id, name").Where("id = ?", clientID).First(&client).Error; err != nil {
		return "", err
	}
	newSecret, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	if err := db.Model(model.Client{}).Where("id = ?", client.Id).Update("sub_secret", newSecret.String()).Error; err != nil {
		return "", err
	}
	return client.Name, nil
}
