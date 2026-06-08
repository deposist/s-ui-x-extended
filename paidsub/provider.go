package paidsub

import (
	"context"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
)

// ProviderKind identifies a payment backend.
type ProviderKind string

const (
	ProviderStars     ProviderKind = "stars"
	ProviderYooKassa  ProviderKind = "yookassa"
	ProviderStripe    ProviderKind = "stripe"
	ProviderPayMaster ProviderKind = "paymaster"
	ProviderCryptoBot ProviderKind = "cryptobot"
	ProviderExternal  ProviderKind = "external"
)

// InvoiceMethod tells the bot how to deliver an invoice to the user.
type InvoiceMethod int

const (
	InvoiceTelegramNative InvoiceMethod = iota // bot calls sendInvoice
	InvoiceURL                                 // bot sends a pay link
	InvoiceManualLink                          // external link + "I paid" button
)

type LabeledPrice struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// Invoice is the provider-agnostic result of preparing a payment.
type Invoice struct {
	Method        InvoiceMethod
	Title         string
	Description   string
	ProviderToken string // "" for Stars
	Currency      string // "XTR" for Stars
	Prices        []LabeledPrice
	Payload       string // == order.IdempotencyKey, echoed back by Telegram
	PayURL        string // for InvoiceURL / InvoiceManualLink
	ProviderRef   string // provider-side id to persist (e.g. cryptobot invoice_id)
}

// PollResult reports an out-of-band confirmed payment.
type PollResult struct {
	OrderID          uint
	ProviderChargeID string
	RawPayload       []byte
}

// PaymentProvider prepares invoices and declares how it confirms.
type PaymentProvider interface {
	Kind() ProviderKind
	Title(l lang) string
	CreateInvoice(ctx context.Context, order *PaymentOrder, tariff *Tariff, client *model.Client) (*Invoice, error)
}

// pollingProvider is implemented by providers confirmed via polling (CryptoBot).
type pollingProvider interface {
	Poll(ctx context.Context, pending []PaymentOrder) ([]PollResult, error)
}

func providerTitle(kind ProviderKind, l lang) string {
	switch kind {
	case ProviderStars:
		return "⭐ Telegram Stars"
	case ProviderYooKassa:
		return "💳 YooKassa"
	case ProviderStripe:
		return "💳 Stripe"
	case ProviderPayMaster:
		return "💳 PayMaster"
	case ProviderCryptoBot:
		return "🪙 CryptoBot"
	case ProviderExternal:
		if l == langRU {
			return "🌐 Оплата по ссылке"
		}
		return "🌐 External link"
	}
	return string(kind)
}

// ---- Telegram-native provider (Stars / YooKassa / Stripe) ----

type telegramProvider struct {
	kind  ProviderKind
	token string // provider_token (empty for Stars)
}

func (p *telegramProvider) Kind() ProviderKind  { return p.kind }
func (p *telegramProvider) Title(l lang) string { return providerTitle(p.kind, l) }

func (p *telegramProvider) CreateInvoice(ctx context.Context, order *PaymentOrder, tariff *Tariff, client *model.Client) (*Invoice, error) {
	desc := tariff.Description
	if strings.TrimSpace(desc) == "" {
		desc = tariff.Name
	}
	inv := &Invoice{
		Method:      InvoiceTelegramNative,
		Title:       tariff.Name,
		Description: desc,
		Payload:     order.IdempotencyKey,
	}
	if p.kind == ProviderStars {
		inv.Currency = "XTR"
		inv.ProviderToken = ""
		inv.Prices = []LabeledPrice{{Label: tariff.Name, Amount: tariff.StarsAmount}}
	} else {
		inv.Currency = order.Currency
		inv.ProviderToken = p.token
		inv.Prices = []LabeledPrice{{Label: tariff.Name, Amount: tariff.Price}}
	}
	return inv, nil
}

// ---- External link provider ----

type externalProvider struct {
	template string
}

func (p *externalProvider) Kind() ProviderKind  { return ProviderExternal }
func (p *externalProvider) Title(l lang) string { return providerTitle(ProviderExternal, l) }

func (p *externalProvider) CreateInvoice(ctx context.Context, order *PaymentOrder, tariff *Tariff, client *model.Client) (*Invoice, error) {
	url := renderExternalURL(p.template, order, tariff, client)
	return &Invoice{
		Method:  InvoiceManualLink,
		Title:   tariff.Name,
		PayURL:  url,
		Payload: order.IdempotencyKey,
	}, nil
}

func renderExternalURL(tmpl string, order *PaymentOrder, tariff *Tariff, client *model.Client) string {
	r := strings.NewReplacer(
		"{orderId}", strconv.FormatUint(uint64(order.Id), 10),
		"{clientId}", strconv.FormatUint(uint64(client.Id), 10),
		"{tariffId}", strconv.FormatUint(uint64(tariff.Id), 10),
		"{amount}", strconv.FormatInt(order.Amount, 10),
		"{currency}", order.Currency,
		"{key}", order.IdempotencyKey,
	)
	return r.Replace(tmpl)
}
