// Package paidsub implements the experimental "Paid Subscriptions" module: a
// client-facing Telegram bot (subscription links, QR codes, usage stats),
// self-registration with a trial period, and tariff-based payments through
// multiple providers. It is isolated from the core: wired in at app/api start
// behind the paidSubEnabled flag and owns its own DB schema (see schema.go).
package paidsub

// Tariff is an admin-defined purchasable plan. Price is in minor currency units
// (e.g. kopeks/cents); StarsAmount is the whole-number Telegram Stars (XTR)
// price used when paying via Stars. AddDays extends the client's Expiry and
// AddTrafficBytes refills Volume on a successful payment.
type Tariff struct {
	Id              uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Price           int64  `json:"price" gorm:"not null;default:0"`
	Currency        string `json:"currency" gorm:"not null;default:RUB"`
	StarsAmount     int64  `json:"starsAmount" gorm:"column:stars_amount;not null;default:0"`
	AddDays         int    `json:"addDays" gorm:"column:add_days;not null;default:0"`
	AddTrafficBytes int64  `json:"addTrafficBytes" gorm:"column:add_traffic_bytes;not null;default:0"`
	Sort            int    `json:"sort" gorm:"not null;default:0"`
	Enabled         bool   `json:"enabled" gorm:"not null;default:true"`
	CreatedAt       int64  `json:"createdAt" gorm:"column:created_at;not null;default:0"`
	UpdatedAt       int64  `json:"updatedAt" gorm:"column:updated_at;not null;default:0"`
}

func (Tariff) TableName() string { return "tariffs" }

// PaymentOrder is one purchase attempt. Amount/Currency are a server-side
// snapshot of the tariff at order-creation time (never trusted from the client
// or the payment update). Status transitions out of "pending" are terminal and
// guarded so a renewal applies at most once.
type PaymentOrder struct {
	Id             uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId       uint   `json:"clientId" gorm:"column:client_id;index;not null"`
	TariffId       uint   `json:"tariffId" gorm:"column:tariff_id;index;not null"`
	Provider       string `json:"provider" gorm:"index;not null"`
	Amount         int64  `json:"amount" gorm:"not null;default:0"`
	Currency       string `json:"currency" gorm:"not null"`
	Status         string `json:"status" gorm:"index;not null;default:pending"`
	TelegramUserId int64  `json:"telegramUserId" gorm:"column:telegram_user_id;index;not null;default:0"`
	// IdempotencyKey (the Telegram invoice payload) and ProviderChargeID are
	// internal provider ids — never serialized to the browser (json:"-"), so the
	// admin Orders API exposes no provider secrets/ids (mirrors ProviderPayload).
	IdempotencyKey   string `json:"-" gorm:"column:idempotency_key;uniqueIndex;not null"`
	ProviderChargeID string `json:"-" gorm:"column:provider_charge_id;index"`
	ProviderPayload  []byte `json:"-" gorm:"column:provider_payload"`
	ExternalURL      string `json:"externalUrl" gorm:"column:external_url"`
	CreatedAt        int64  `json:"createdAt" gorm:"column:created_at;index;not null;default:0"`
	PaidAt           int64  `json:"paidAt" gorm:"column:paid_at;not null;default:0"`
	ExpiresAt        int64  `json:"expiresAt" gorm:"column:expires_at;index;not null;default:0"`
}

func (PaymentOrder) TableName() string { return "payment_orders" }

// Binding maps a Telegram user to exactly one client in the core `clients`
// table. Both directions are UNIQUE: one Telegram user ↔ one client. Rows exist
// only for explicitly bound users, so there is no default-0 collision risk and
// the core `clients` table is left untouched.
type Binding struct {
	Id        uint  `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId  uint  `json:"clientId" gorm:"column:client_id;uniqueIndex;not null"`
	TgUserId  int64 `json:"tgUserId" gorm:"column:tg_user_id;uniqueIndex;not null"`
	CreatedAt int64 `json:"createdAt" gorm:"column:created_at;not null;default:0"`
	UpdatedAt int64 `json:"updatedAt" gorm:"column:updated_at;not null;default:0"`
}

func (Binding) TableName() string { return "paidsub_bindings" }

// Order status constants.
const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusFailed   = "failed"
	StatusExpired  = "expired"
	StatusRefunded = "refunded"
)
