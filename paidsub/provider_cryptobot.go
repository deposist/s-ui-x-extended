package paidsub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
)

// cryptoBotBase is pinned (never configurable) to prevent token exfiltration.
const cryptoBotBase = "https://pay.crypt.bot"

const (
	cryptoBotPollBatchSize     = 100
	cryptoBotReconcilePageSize = 100
	cryptoBotReconcileMaxPages = 20
)

type cryptoBotHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type cryptoBotProvider struct {
	token   string
	baseURL string
	client  cryptoBotHTTPDoer
}

type cryptoBotInvoice struct {
	InvoiceID    json.Number `json:"invoice_id"`
	Status       string      `json:"status"`
	Amount       string      `json:"amount"`
	CurrencyType string      `json:"currency_type"`
	Fiat         string      `json:"fiat"`
	Payload      string      `json:"payload"`
	PayURL       string      `json:"bot_invoice_url"`
	LegacyURL    string      `json:"pay_url"`
}

type cryptoBotAPIError struct {
	Code int
	Name string
}

func (e *cryptoBotAPIError) Error() string {
	if e.Name != "" {
		return "cryptobot: api error " + e.Name
	}
	return fmt.Sprintf("cryptobot: api error %d", e.Code)
}

func (p *cryptoBotProvider) Kind() ProviderKind  { return ProviderCryptoBot }
func (p *cryptoBotProvider) Title(l lang) string { return providerTitle(ProviderCryptoBot, l) }

func (p *cryptoBotProvider) CreateInvoice(ctx context.Context, order *PaymentOrder, tariff *Tariff, client *model.Client) (*Invoice, error) {
	amount := cryptoBotFormatAmount(order.Amount)
	body := map[string]any{
		"currency_type": "fiat",
		"fiat":          order.Currency,
		"amount":        amount,
		"payload":       order.IdempotencyKey,
		"description":   tariff.Name,
	}
	var out struct {
		InvoiceID json.Number `json:"invoice_id"`
		PayURL    string      `json:"pay_url"`
	}
	if err := p.call(ctx, http.MethodPost, "/api/createInvoice", body, &out); err != nil {
		return nil, err
	}
	if !validCryptoBotInvoiceRef(out.InvoiceID.String()) || out.PayURL == "" {
		return nil, fmt.Errorf("cryptobot: incomplete invoice response")
	}
	return &Invoice{
		Method:      InvoiceURL,
		Title:       tariff.Name,
		PayURL:      out.PayURL,
		ProviderRef: out.InvoiceID.String(),
		Payload:     order.IdempotencyKey,
	}, nil
}

func (p *cryptoBotProvider) DeleteInvoice(ctx context.Context, providerRef string) error {
	invoiceID, err := strconv.ParseInt(providerRef, 10, 64)
	if err != nil || invoiceID <= 0 {
		return fmt.Errorf("cryptobot: invalid invoice reference")
	}
	err = p.call(ctx, http.MethodPost, "/api/deleteInvoice", map[string]any{"invoice_id": invoiceID}, nil)
	if isCryptoBotInvoiceTerminalError(err) {
		return nil
	}
	return err
}

func (p *cryptoBotProvider) Poll(ctx context.Context, pending []PaymentOrder) (PollOutcome, error) {
	if len(pending) == 0 {
		return PollOutcome{}, nil
	}
	if len(pending) > cryptoBotPollBatchSize {
		return PollOutcome{}, fmt.Errorf("cryptobot: poll batch exceeds %d orders", cryptoBotPollBatchSize)
	}
	idToOrder := make(map[string]PaymentOrder, len(pending))
	ids := make([]string, 0, len(pending))
	for _, order := range pending {
		ref := order.ProviderRef
		if ref == "" {
			ref = extractProviderRef(order.ProviderPayload)
		}
		if !validCryptoBotInvoiceRef(ref) {
			continue
		}
		if other, exists := idToOrder[ref]; exists && other.Id != order.Id {
			return PollOutcome{}, fmt.Errorf("cryptobot: ambiguous invoice reference %s", ref)
		}
		idToOrder[ref] = order
		ids = append(ids, ref)
	}
	if len(ids) == 0 {
		return PollOutcome{}, nil
	}
	var out struct {
		Items []cryptoBotInvoice `json:"items"`
	}
	path := "/api/getInvoices?invoice_ids=" + url.QueryEscape(strings.Join(ids, ","))
	if err := p.call(ctx, http.MethodGet, path, nil, &out); err != nil {
		return PollOutcome{}, err
	}
	outcome := PollOutcome{}
	seen := make(map[string]struct{}, len(out.Items))
	for _, invoice := range out.Items {
		invoiceID := invoice.InvoiceID.String()
		order, ok := idToOrder[invoiceID]
		if !ok {
			continue
		}
		if _, duplicate := seen[invoiceID]; duplicate {
			return PollOutcome{}, fmt.Errorf("cryptobot: duplicate invoice %s in poll response", invoiceID)
		}
		seen[invoiceID] = struct{}{}
		if !cryptoBotInvoiceMatches(order, invoice) {
			logger.Warning("paidsub: cryptobot invoice metadata mismatch; refusing order ", order.Id)
			(&service.TelegramService{}).NotifyTelegramEvent("paidsub_payment_mismatch", map[string]string{
				"orderId": fmt.Sprintf("%d", order.Id),
			})
			continue
		}
		switch strings.ToLower(invoice.Status) {
		case "paid":
			raw, _ := json.Marshal(invoice)
			outcome.Paid = append(outcome.Paid, PollResult{
				OrderID:          order.Id,
				ProviderChargeID: "cryptobot:" + invoiceID,
				RawPayload:       raw,
			})
		case "expired", "deleted":
			outcome.TerminalOrderIDs = append(outcome.TerminalOrderIDs, order.Id)
		}
	}
	for invoiceID, order := range idToOrder {
		if _, ok := seen[invoiceID]; !ok {
			outcome.MissingOrderIDs = append(outcome.MissingOrderIDs, order.Id)
		}
	}
	return outcome, nil
}

func (p *cryptoBotProvider) ReconcileInvoices(ctx context.Context, unresolved []PaymentOrder) ([]ReconciledInvoice, error) {
	byPayload := make(map[string]PaymentOrder, len(unresolved))
	for _, order := range unresolved {
		if order.IdempotencyKey == "" {
			return nil, fmt.Errorf("cryptobot: unresolved order %d has empty payload", order.Id)
		}
		if other, exists := byPayload[order.IdempotencyKey]; exists && other.Id != order.Id {
			return nil, fmt.Errorf("cryptobot: ambiguous order payload")
		}
		byPayload[order.IdempotencyKey] = order
	}
	if len(byPayload) == 0 {
		return nil, nil
	}

	reconciled := make([]ReconciledInvoice, 0, len(unresolved))
	found := make(map[uint]bool, len(unresolved))
	seenRefs := make(map[string]struct{})
	for page := 0; page < cryptoBotReconcileMaxPages; page++ {
		offset := page * cryptoBotReconcilePageSize
		var out struct {
			Items []cryptoBotInvoice `json:"items"`
		}
		path := fmt.Sprintf("/api/getInvoices?offset=%d&count=%d", offset, cryptoBotReconcilePageSize)
		if err := p.call(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		for _, invoice := range out.Items {
			order, ok := byPayload[invoice.Payload]
			if !ok {
				continue
			}
			ref := invoice.InvoiceID.String()
			if !validCryptoBotInvoiceRef(ref) {
				return nil, fmt.Errorf("cryptobot: invalid invoice reference in reconciliation")
			}
			if _, duplicate := seenRefs[ref]; duplicate {
				return nil, fmt.Errorf("cryptobot: repeated invoice page")
			}
			seenRefs[ref] = struct{}{}
			payURL := invoice.PayURL
			if payURL == "" {
				payURL = invoice.LegacyURL
			}
			found[order.Id] = true
			reconciled = append(reconciled, ReconciledInvoice{
				OrderID:          order.Id,
				ProviderRef:      ref,
				PayURL:           payURL,
				Paid:             strings.EqualFold(invoice.Status, "paid"),
				ProviderStatus:   strings.ToLower(invoice.Status),
				MetadataMismatch: !cryptoBotInvoiceMatches(order, invoice),
				ProviderChargeID: "cryptobot:" + ref,
			})
		}
		if len(out.Items) < cryptoBotReconcilePageSize {
			for _, order := range unresolved {
				if !found[order.Id] {
					reconciled = append(reconciled, ReconciledInvoice{OrderID: order.Id, ProviderStatus: "missing"})
				}
			}
			return reconciled, nil
		}
	}
	return nil, fmt.Errorf("cryptobot: reconciliation exceeded %d pages", cryptoBotReconcileMaxPages)
}

func cryptoBotInvoiceMatches(order PaymentOrder, invoice cryptoBotInvoice) bool {
	return invoice.Payload != "" && invoice.Payload == order.IdempotencyKey &&
		invoice.CurrencyType == "fiat" && invoice.Fiat != "" && invoice.Fiat == order.Currency &&
		cryptoBotAmountMatches(invoice.Amount, order.Amount)
}

func cryptoBotFormatAmount(minor int64) string {
	return fmt.Sprintf("%d.%02d", minor/100, minor%100)
}

func cryptoBotAmountMatches(amount string, minor int64) bool {
	if amount == "" || minor < 0 || strings.TrimSpace(amount) != amount {
		return false
	}
	parts := strings.Split(amount, ".")
	if len(parts) > 2 || parts[0] == "" {
		return false
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return false
		}
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return false
		}
		for _, r := range fraction {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > (int64(^uint64(0)>>1)-99)/100 {
		return false
	}
	frac := int64(0)
	if fraction != "" {
		frac, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return false
		}
	}
	return whole*100+frac == minor
}

func validCryptoBotInvoiceRef(ref string) bool {
	id, err := strconv.ParseInt(ref, 10, 64)
	return err == nil && id > 0
}

func isCryptoBotInvoiceTerminalError(err error) bool {
	var apiErr *cryptoBotAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	name := strings.ToUpper(apiErr.Name)
	return strings.Contains(name, "INVOICE_NOT_FOUND") ||
		strings.Contains(name, "INVOICE_ALREADY_DELETED") ||
		strings.Contains(name, "INVOICE_DELETED")
}

// call performs a CryptoBot API request. The API token is sent in a header
// (never the URL) and is never logged; errors carry no request details.
func (p *cryptoBotProvider) call(ctx context.Context, method, path string, body any, out any) error {
	client := p.client
	if client == nil {
		configured, err := service.NewPaidSubHTTPClient(15 * time.Second)
		if err != nil {
			return err
		}
		client = configured
	}
	var reader io.Reader
	if body != nil {
		bb, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(bb)
	}
	baseURL := p.baseURL
	if baseURL == "" {
		baseURL = cryptoBotBase
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Crypto-Pay-API-Token", p.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("cryptobot: network error")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes+1))
	if err != nil {
		return fmt.Errorf("cryptobot: response read error")
	}
	if len(data) > maxTelegramResponseBytes {
		return fmt.Errorf("cryptobot: response too large")
	}
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  struct {
			Code int    `json:"code"`
			Name string `json:"name"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("cryptobot: malformed response")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if !env.OK {
			return &cryptoBotAPIError{Code: env.Error.Code, Name: env.Error.Name}
		}
		return fmt.Errorf("cryptobot: http status %d", resp.StatusCode)
	}
	if !env.OK {
		return &cryptoBotAPIError{Code: env.Error.Code, Name: env.Error.Name}
	}
	if out != nil {
		if len(env.Result) == 0 || string(env.Result) == "null" {
			return fmt.Errorf("cryptobot: response missing result")
		}
		if err := json.Unmarshal(env.Result, out); err != nil {
			return fmt.Errorf("cryptobot: malformed result")
		}
	}
	return nil
}

func extractProviderRef(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	var m struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(payload, &m); err != nil {
		return ""
	}
	return m.Ref
}
