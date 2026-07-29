// Package paidsub implements the experimental "Paid Subscriptions" module: a
// client-facing Telegram bot, self-registration, and tariff-based payments.
package paidsub

import "github.com/deposist/s-ui-x-extended/database/model"

// Persistent PaidSub models live in database/model so central backup and
// restore can include them without importing paidsub and creating a package
// cycle. Aliases preserve the existing paidsub API.
type Tariff = model.PaidSubTariff
type PaymentOrder = model.PaidSubPaymentOrder
type Binding = model.PaidSubBinding

// Order status constants.
const (
	StatusPending         = "pending"
	StatusInvoiceCreating = "invoice_creating"
	StatusRecoverable     = "recoverable"
	StatusPaid            = "paid"
	StatusFailed          = "failed"
	StatusExpired         = "expired"
	StatusRefunded        = "refunded"
)
