package paidsub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
)

// botAPIBase is pinned (never configurable) to avoid token exfiltration via a
// rogue host. The bot token lives in the URL path, so request URLs and error
// bodies must never be logged.
const botAPIBase = "https://api.telegram.org"

const maxTelegramResponseBytes = 1 << 20

// tgAPIError carries Telegram's error_code/description WITHOUT the request URL
// (which contains the bot token). Safe to log.
type tgAPIError struct {
	Method      string
	Code        int
	Description string
	RetryAfter  int
}

func (e *tgAPIError) Error() string {
	return fmt.Sprintf("telegram %s failed: code=%d %s", e.Method, e.Code, e.Description)
}

func (b *Bot) apiURL(method string) string {
	return botAPIBase + "/bot" + b.token + "/" + method
}

func (b *Bot) callJSON(ctx context.Context, method string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiURL(method), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		// Do not wrap: the underlying error may embed the request URL (token).
		return fmt.Errorf("telegram %s: network error", method)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	_, err = parseTelegramResponse(method, data)
	return err
}

func parseTelegramResponse(method string, data []byte) (json.RawMessage, error) {
	var tr tgResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, fmt.Errorf("telegram %s: malformed response", method)
	}
	if !tr.OK {
		apiErr := &tgAPIError{Method: method, Code: tr.ErrorCode, Description: tr.Description}
		if tr.Parameters != nil {
			apiErr.RetryAfter = tr.Parameters.RetryAfter
		}
		return nil, apiErr
	}
	return tr.Result, nil
}

// getUpdates long-polls. The bot's HTTP client carries a timeout > the poll
// interval so the long poll completes server-side.
func (b *Bot) getUpdates(ctx context.Context, offset int64, timeout int) ([]tgUpdate, error) {
	u := b.apiURL("getUpdates") +
		"?timeout=" + strconv.Itoa(timeout) +
		"&offset=" + strconv.FormatInt(offset, 10) +
		"&allowed_updates=" + url.QueryEscape(`["message","callback_query","pre_checkout_query"]`)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getUpdates: network error")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	result, err := parseTelegramResponse("getUpdates", data)
	if err != nil {
		return nil, err
	}
	var updates []tgUpdate
	if err := json.Unmarshal(result, &updates); err != nil {
		return nil, fmt.Errorf("telegram getUpdates: malformed result")
	}
	return updates, nil
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string, markup *inlineKeyboard) error {
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	return b.callJSON(ctx, "sendMessage", payload)
}

func (b *Bot) answerCallback(ctx context.Context, callbackID string, text string) error {
	payload := map[string]any{"callback_query_id": callbackID}
	if text != "" {
		payload["text"] = text
	}
	return b.callJSON(ctx, "answerCallbackQuery", payload)
}

func (b *Bot) sendInvoice(ctx context.Context, chatID int64, inv *Invoice) error {
	payload := map[string]any{
		"chat_id":     chatID,
		"title":       inv.Title,
		"description": inv.Description,
		"payload":     inv.Payload,
		"currency":    inv.Currency,
		"prices":      inv.Prices,
	}
	if inv.ProviderToken != "" {
		payload["provider_token"] = inv.ProviderToken
	}
	return b.callJSON(ctx, "sendInvoice", payload)
}

// refundStarPayment refunds a successful Telegram Stars (XTR) payment. chargeID
// is the raw telegram_payment_charge_id (without the stored "tg:" prefix). Only
// Stars payments can be refunded through the Bot API; fiat/crypto refunds are
// handled out-of-band.
func (b *Bot) refundStarPayment(ctx context.Context, userID int64, chargeID string) error {
	return b.callJSON(ctx, "refundStarPayment", map[string]any{
		"user_id":                    userID,
		"telegram_payment_charge_id": chargeID,
	})
}

func (b *Bot) answerPreCheckout(ctx context.Context, queryID string, ok bool, errMsg string) error {
	payload := map[string]any{
		"pre_checkout_query_id": queryID,
		"ok":                    ok,
	}
	if !ok && errMsg != "" {
		payload["error_message"] = errMsg
	}
	return b.callJSON(ctx, "answerPreCheckoutQuery", payload)
}

func (b *Bot) sendPhoto(ctx context.Context, chatID int64, png []byte, caption string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return err
	}
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return err
		}
	}
	part, err := w.CreateFormFile("photo", "qr.png")
	if err != nil {
		return err
	}
	if _, err := part.Write(png); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiURL("sendPhoto"), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendPhoto: network error")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendPhoto: status %d", resp.StatusCode)
	}
	return nil
}
