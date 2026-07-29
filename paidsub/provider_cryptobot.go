package paidsub

import (
	"bytes"
	"context"
	"encoding/json"
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

const cryptoBotReconcilePageSize = 1000

type cryptoBotHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type cryptoBotProvider struct {
	token   string
	baseURL string
	client  cryptoBotHTTPDoer
}

func (p *cryptoBotProvider) Kind() ProviderKind  { return ProviderCryptoBot }
func (p *cryptoBotProvider) Title(l lang) string { return providerTitle(ProviderCryptoBot, l) }

func (p *cryptoBotProvider) CreateInvoice(ctx context.Context, order *PaymentOrder, tariff *Tariff, client *model.Client) (*Invoice, error) {
	amount := fmt.Sprintf("%.2f", float64(order.Amount)/100.0)
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
	return p.call(ctx, http.MethodPost, "/api/deleteInvoice", map[string]any{"invoice_id": invoiceID}, nil)
}

func (p *cryptoBotProvider) Poll(ctx context.Context, pending []PaymentOrder) ([]PollResult, error) {
	idToOrder := map[string]PaymentOrder{}
	var ids []string
	for _, o := range pending {
		ref := o.ProviderRef
		if ref == "" {
			ref = extractProviderRef(o.ProviderPayload)
		}
		if ref == "" {
			continue
		}
		idToOrder[ref] = o
		ids = append(ids, ref)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var out struct {
		Items []struct {
			InvoiceID json.Number `json:"invoice_id"`
			Status    string      `json:"status"`
			Amount    string      `json:"amount"`
			Fiat      string      `json:"fiat"`
		} `json:"items"`
	}
	path := "/api/getInvoices?invoice_ids=" + url.QueryEscape(strings.Join(ids, ","))
	if err := p.call(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	var results []PollResult
	for _, it := range out.Items {
		if it.Status != "paid" {
			continue
		}
		invID := it.InvoiceID.String()
		order, ok := idToOrder[invID]
		if !ok {
			continue
		}
		// Defense in depth: re-validate the paid amount/currency against the
		// server-side order snapshot before granting (mirrors the Stars path).
		// The invoice amount is server-fixed at creation, so a mismatch is
		// anomalous - refuse and alert. Fail OPEN when the provider omits the
		// amount field so a response-format change never blocks a real payment.
		if it.Amount != "" {
			want := fmt.Sprintf("%.2f", float64(order.Amount)/100.0)
			got := it.Amount
			if paid, perr := strconv.ParseFloat(it.Amount, 64); perr == nil {
				got = fmt.Sprintf("%.2f", paid)
			}
			currencyMismatch := it.Fiat != "" && !strings.EqualFold(it.Fiat, order.Currency)
			if got != want || currencyMismatch {
				logger.Warning("paidsub: cryptobot paid amount/currency mismatch; refusing order ", order.Id)
				(&service.TelegramService{}).NotifyTelegramEvent("paidsub_payment_mismatch", map[string]string{
					"orderId": fmt.Sprintf("%d", order.Id),
				})
				continue
			}
		}
		results = append(results, PollResult{
			OrderID:          order.Id,
			ProviderChargeID: "cryptobot:" + invID,
		})
	}
	return results, nil
}

func (p *cryptoBotProvider) ReconcileInvoices(ctx context.Context, unresolved []PaymentOrder) ([]ReconciledInvoice, error) {
	byPayload := make(map[string]PaymentOrder, len(unresolved))
	for _, order := range unresolved {
		byPayload[order.IdempotencyKey] = order
	}
	if len(byPayload) == 0 {
		return nil, nil
	}

	var reconciled []ReconciledInvoice
	for offset := 0; ; offset += cryptoBotReconcilePageSize {
		var out struct {
			Items []struct {
				InvoiceID json.Number `json:"invoice_id"`
				Status    string      `json:"status"`
				Amount    string      `json:"amount"`
				Fiat      string      `json:"fiat"`
				Payload   string      `json:"payload"`
				PayURL    string      `json:"bot_invoice_url"`
				LegacyURL string      `json:"pay_url"`
			} `json:"items"`
		}
		path := fmt.Sprintf("/api/getInvoices?offset=%d&count=%d", offset, cryptoBotReconcilePageSize)
		if err := p.call(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		for _, invoice := range out.Items {
			order, ok := byPayload[invoice.Payload]
			if !ok || !cryptoBotInvoiceMatches(order, invoice.Amount, invoice.Fiat) {
				continue
			}
			payURL := invoice.PayURL
			if payURL == "" {
				payURL = invoice.LegacyURL
			}
			ref := invoice.InvoiceID.String()
			reconciled = append(reconciled, ReconciledInvoice{
				OrderID:          order.Id,
				ProviderRef:      ref,
				PayURL:           payURL,
				Paid:             invoice.Status == "paid",
				ProviderChargeID: "cryptobot:" + ref,
			})
			delete(byPayload, invoice.Payload)
		}
		if len(out.Items) < cryptoBotReconcilePageSize || len(byPayload) == 0 {
			return reconciled, nil
		}
	}
}

func cryptoBotInvoiceMatches(order PaymentOrder, amount, fiat string) bool {
	want := fmt.Sprintf("%.2f", float64(order.Amount)/100.0)
	got := amount
	if paid, err := strconv.ParseFloat(amount, 64); err == nil {
		got = fmt.Sprintf("%.2f", paid)
	}
	return got == want && strings.EqualFold(fiat, order.Currency)
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
		bb, mErr := json.Marshal(body)
		if mErr != nil {
			return mErr
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
		return fmt.Errorf("cryptobot: network error")
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("cryptobot: malformed response")
	}
	if !env.OK {
		return fmt.Errorf("cryptobot: api returned not-ok")
	}
	if out != nil && len(env.Result) > 0 {
		return json.Unmarshal(env.Result, out)
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
