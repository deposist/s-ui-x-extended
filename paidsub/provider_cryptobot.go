package paidsub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/deposist/s-ui-x/service"
)

// cryptoBotBase is pinned (never configurable) to prevent token exfiltration.
const cryptoBotBase = "https://pay.crypt.bot"

type cryptoBotProvider struct {
	token string
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

func (p *cryptoBotProvider) Poll(ctx context.Context, pending []PaymentOrder) ([]PollResult, error) {
	idToOrder := map[string]uint{}
	var ids []string
	for _, o := range pending {
		ref := extractProviderRef(o.ProviderPayload)
		if ref == "" {
			continue
		}
		idToOrder[ref] = o.Id
		ids = append(ids, ref)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var out struct {
		Items []struct {
			InvoiceID json.Number `json:"invoice_id"`
			Status    string      `json:"status"`
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
		if oid, ok := idToOrder[invID]; ok {
			results = append(results, PollResult{
				OrderID:          oid,
				ProviderChargeID: "cryptobot:" + invID,
			})
		}
	}
	return results, nil
}

// call performs a CryptoBot API request. The API token is sent in a header
// (never the URL) and is never logged; errors carry no request details.
func (p *cryptoBotProvider) call(ctx context.Context, method, path string, body any, out any) error {
	client, err := service.NewPaidSubHTTPClient(15 * time.Second)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		bb, mErr := json.Marshal(body)
		if mErr != nil {
			return mErr
		}
		reader = bytes.NewReader(bb)
	}
	req, err := http.NewRequestWithContext(ctx, method, cryptoBotBase+path, reader)
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
