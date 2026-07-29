package paidsub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCryptoBotReconcileFindsInvoiceByPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/getInvoices" || r.URL.Query().Get("count") != "1000" {
			t.Fatalf("unexpected reconciliation request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"paid","amount":"1.00","fiat":"RUB","payload":"wanted","bot_invoice_url":"https://pay.example/123"},` +
			`{"invoice_id":456,"status":"active","amount":"9.99","fiat":"RUB","payload":"other"}` +
			`]}}`))
	}))
	defer server.Close()

	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	results, err := provider.ReconcileInvoices(context.Background(), []PaymentOrder{{
		Id: 7, IdempotencyKey: "wanted", Amount: 100, Currency: "RUB",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].OrderID != 7 || results[0].ProviderRef != "123" || !results[0].Paid || results[0].PayURL == "" {
		t.Fatalf("unexpected reconciled invoices: %+v", results)
	}
}

func TestCryptoBotReconcileRejectsAmountOrCurrencyMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"paid","amount":"2.00","fiat":"RUB","payload":"wanted"},` +
			`{"invoice_id":124,"status":"paid","amount":"1.00","fiat":"USD","payload":"wanted"}` +
			`]}}`))
	}))
	defer server.Close()

	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	results, err := provider.ReconcileInvoices(context.Background(), []PaymentOrder{{
		Id: 7, IdempotencyKey: "wanted", Amount: 100, Currency: "RUB",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("mismatched invoice was reconciled: %+v", results)
	}
}
