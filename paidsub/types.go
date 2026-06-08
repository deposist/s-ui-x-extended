package paidsub

import "encoding/json"

// Minimal Telegram Bot API types — only the fields this module consumes. The
// bot trusts only updates fetched from the official API over its own token.

type tgUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LanguageCode string `json:"language_code"`
}

type tgChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type tgSuccessfulPayment struct {
	Currency                string `json:"currency"`
	TotalAmount             int64  `json:"total_amount"`
	InvoicePayload          string `json:"invoice_payload"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
}

type tgMessage struct {
	MessageID         int64                `json:"message_id"`
	From              *tgUser              `json:"from"`
	Chat              tgChat               `json:"chat"`
	Text              string               `json:"text"`
	SuccessfulPayment *tgSuccessfulPayment `json:"successful_payment"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	From    tgUser     `json:"from"`
	Data    string     `json:"data"`
	Message *tgMessage `json:"message"`
}

type tgPreCheckoutQuery struct {
	ID             string `json:"id"`
	From           tgUser `json:"from"`
	Currency       string `json:"currency"`
	TotalAmount    int64  `json:"total_amount"`
	InvoicePayload string `json:"invoice_payload"`
}

type tgUpdate struct {
	UpdateID         int64               `json:"update_id"`
	Message          *tgMessage          `json:"message"`
	CallbackQuery    *tgCallbackQuery    `json:"callback_query"`
	PreCheckoutQuery *tgPreCheckoutQuery `json:"pre_checkout_query"`
}

type tgResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

// Inline keyboard.
type inlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
	Pay          bool   `json:"pay,omitempty"`
}

type inlineKeyboard struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}
