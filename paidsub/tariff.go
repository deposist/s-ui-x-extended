package paidsub

import (
	"encoding/json"

	"github.com/deposist/s-ui-x/database"

	"gorm.io/gorm"
)

// TariffService is the admin CRUD + bot-read service for tariffs. It is scoped
// to the tariffs table only.
type TariffService struct{}

func NewTariffService() *TariffService { return &TariffService{} }

func (s *TariffService) GetAll() ([]Tariff, error) {
	db := database.GetDB()
	var tariffs []Tariff
	if err := db.Order("sort asc, id asc").Find(&tariffs).Error; err != nil {
		return nil, err
	}
	return tariffs, nil
}

// GetEnabled returns enabled tariffs, ordered for display in the bot.
func (s *TariffService) GetEnabled() ([]Tariff, error) {
	db := database.GetDB()
	var tariffs []Tariff
	if err := db.Where("enabled = ?", true).Order("sort asc, id asc").Find(&tariffs).Error; err != nil {
		return nil, err
	}
	return tariffs, nil
}

func (s *TariffService) Get(id uint) (*Tariff, error) {
	db := database.GetDB()
	var t Tariff
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Save applies a CRUD action ("new" | "edit" | "del" | "delbulk").
func (s *TariffService) Save(act string, data json.RawMessage) error {
	db := database.GetDB()
	now := nowUnix()
	switch act {
	case "new":
		var t Tariff
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		t.Id = 0
		t.CreatedAt = now
		t.UpdatedAt = now
		return db.Create(&t).Error
	case "edit":
		var t Tariff
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		if t.Id == 0 {
			return gorm.ErrMissingWhereClause
		}
		t.UpdatedAt = now
		// Explicit column list so zero-valued fields (price=0, enabled=false)
		// are persisted and CreatedAt is preserved.
		return db.Model(&Tariff{}).Where("id = ?", t.Id).Updates(map[string]any{
			"name":              t.Name,
			"description":       t.Description,
			"price":             t.Price,
			"currency":          t.Currency,
			"stars_amount":      t.StarsAmount,
			"add_days":          t.AddDays,
			"add_traffic_bytes": t.AddTrafficBytes,
			"sort":              t.Sort,
			"enabled":           t.Enabled,
			"updated_at":        t.UpdatedAt,
		}).Error
	case "del":
		var id uint
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		return db.Where("id = ?", id).Delete(&Tariff{}).Error
	case "delbulk":
		var ids []uint
		if err := json.Unmarshal(data, &ids); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		return db.Where("id IN ?", ids).Delete(&Tariff{}).Error
	default:
		return gorm.ErrInvalidData
	}
}
