package model

// PaidSubTariff is an admin-defined purchasable subscription plan.
type PaidSubTariff struct {
	Id              uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Price           int64  `json:"price" gorm:"not null;default:0"`
	Currency        string `json:"currency" gorm:"not null;default:RUB"`
	StarsAmount     int64  `json:"starsAmount" gorm:"column:stars_amount;not null;default:0"`
	AddDays         int    `json:"addDays" gorm:"column:add_days;not null;default:0"`
	AddTrafficBytes int64  `json:"addTrafficBytes" gorm:"column:add_traffic_bytes;not null;default:0"`
	MaxAWGDevices   int    `json:"maxAwgDevices" gorm:"column:max_awg_devices;not null;default:0"`
	Sort            int    `json:"sort" gorm:"not null;default:0"`
	Enabled         bool   `json:"enabled" gorm:"not null;default:true"`
	CreatedAt       int64  `json:"createdAt" gorm:"column:created_at;not null;default:0"`
	UpdatedAt       int64  `json:"updatedAt" gorm:"column:updated_at;not null;default:0"`
}

func (PaidSubTariff) TableName() string { return "tariffs" }

// PaidSubPaymentOrder stores one purchase attempt and its immutable snapshots.
type PaidSubPaymentOrder struct {
	Id                uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId          uint   `json:"clientId" gorm:"column:client_id;index;not null"`
	TariffId          uint   `json:"tariffId" gorm:"column:tariff_id;index;not null"`
	Provider          string `json:"provider" gorm:"index;not null"`
	Amount            int64  `json:"amount" gorm:"not null;default:0"`
	Currency          string `json:"currency" gorm:"not null"`
	Status            string `json:"status" gorm:"index;not null;default:pending"`
	TelegramUserId    int64  `json:"telegramUserId" gorm:"column:telegram_user_id;index;not null;default:0"`
	IdempotencyKey    string `json:"-" gorm:"column:idempotency_key;uniqueIndex;not null"`
	ProviderChargeID  string `json:"-" gorm:"column:provider_charge_id;index"`
	ProviderPayload   []byte `json:"-" gorm:"column:provider_payload"`
	ExternalURL       string `json:"externalUrl" gorm:"column:external_url"`
	CreatedAt         int64  `json:"createdAt" gorm:"column:created_at;index;not null;default:0"`
	PaidAt            int64  `json:"paidAt" gorm:"column:paid_at;not null;default:0"`
	ExpiresAt         int64  `json:"expiresAt" gorm:"column:expires_at;index;not null;default:0"`
	GrantedUp         int64  `json:"-" gorm:"column:granted_up;not null;default:0"`
	GrantedDown       int64  `json:"-" gorm:"column:granted_down;not null;default:0"`
	GrantedAWGDevices int    `json:"-" gorm:"column:granted_awg_devices;not null;default:0"`
}

func (PaidSubPaymentOrder) TableName() string { return "payment_orders" }

// PaidSubBinding maps one Telegram user to one client.
type PaidSubBinding struct {
	Id        uint  `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId  uint  `json:"clientId" gorm:"column:client_id;uniqueIndex;not null"`
	TgUserId  int64 `json:"tgUserId" gorm:"column:tg_user_id;uniqueIndex;not null"`
	CreatedAt int64 `json:"createdAt" gorm:"column:created_at;not null;default:0"`
	UpdatedAt int64 `json:"updatedAt" gorm:"column:updated_at;not null;default:0"`
}

func (PaidSubBinding) TableName() string { return "paidsub_bindings" }
