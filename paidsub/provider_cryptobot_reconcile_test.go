package paidsub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCryptoBotReconcileFindsInvoiceByPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/getInvoices" || r.URL.Query().Get("count") != "100" {
			t.Fatalf("unexpected reconciliation request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"paid","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"wanted","bot_invoice_url":"https://pay.example/123"},` +
			`{"invoice_id":456,"status":"active","amount":"9.99","currency_type":"fiat","fiat":"RUB","payload":"other"}` +
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

func TestCryptoBotReconcileReportsAmountOrCurrencyMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"paid","amount":"2.00","currency_type":"fiat","fiat":"RUB","payload":"wanted"},` +
			`{"invoice_id":124,"status":"paid","amount":"1.00","currency_type":"fiat","fiat":"USD","payload":"wanted"}` +
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
	if len(results) != 2 || !results[0].MetadataMismatch || !results[1].MetadataMismatch {
		t.Fatalf("mismatched invoices were hidden from reconciliation: %+v", results)
	}
}

func TestCryptoBotReconcileReturnsEveryMatchingInvoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"active","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"duplicate"},` +
			`{"invoice_id":456,"status":"paid","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"duplicate"}` +
			`]}}`))
	}))
	defer server.Close()

	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	results, err := provider.ReconcileInvoices(context.Background(), []PaymentOrder{{
		Id: 7, IdempotencyKey: "duplicate", Amount: 100, Currency: "RUB",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].ProviderRef != "123" || results[0].Paid ||
		results[0].ProviderStatus != "active" || results[1].ProviderRef != "456" || !results[1].Paid ||
		results[1].ProviderStatus != "paid" {
		t.Fatalf("duplicate invoices were not fully reconciled: %+v", results)
	}
}

func TestCryptoBotReconcileReportsExpiredAndMissingInvoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` +
			`{"invoice_id":123,"status":"expired","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"expired"}` +
			`]}}`))
	}))
	defer server.Close()

	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	results, err := provider.ReconcileInvoices(context.Background(), []PaymentOrder{
		{Id: 7, IdempotencyKey: "expired", Amount: 100, Currency: "RUB"},
		{Id: 8, IdempotencyKey: "missing", Amount: 100, Currency: "RUB"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].OrderID != 7 || results[0].ProviderStatus != "expired" ||
		results[1].OrderID != 8 || results[1].ProviderStatus != "missing" {
		t.Fatalf("terminal reconciliation states were lost: %+v", results)
	}
}

func TestCryptoBotPollRequiresCompleteTrustedMetadata(t *testing.T) {
	tests := []struct {
		name    string
		invoice string
	}{
		{name: "missing amount", invoice: `{"invoice_id":123,"status":"paid","currency_type":"fiat","fiat":"RUB","payload":"trusted"}`},
		{name: "wrong amount", invoice: `{"invoice_id":123,"status":"paid","amount":"1.001","currency_type":"fiat","fiat":"RUB","payload":"trusted"}`},
		{name: "missing fiat", invoice: `{"invoice_id":123,"status":"paid","amount":"1.00","currency_type":"fiat","payload":"trusted"}`},
		{name: "wrong currency type", invoice: `{"invoice_id":123,"status":"paid","amount":"1.00","currency_type":"crypto","fiat":"RUB","payload":"trusted"}`},
		{name: "wrong payload", invoice: `{"invoice_id":123,"status":"paid","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"untrusted"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[` + test.invoice + `]}}`))
			}))
			defer server.Close()
			provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
			outcome, err := provider.Poll(context.Background(), []PaymentOrder{{
				Id: 7, ProviderRef: "123", IdempotencyKey: "trusted", Amount: 100, Currency: "RUB",
			}})
			if err != nil {
				t.Fatal(err)
			}
			if len(outcome.Paid) != 0 {
				t.Fatalf("untrusted provider response granted payment: %+v", outcome)
			}
		})
	}
}

func TestCryptoBotPollAcceptsExactFinancialMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"items":[{"invoice_id":123,"status":"paid","amount":"1.00","currency_type":"fiat","fiat":"RUB","payload":"trusted"}]}}`))
	}))
	defer server.Close()
	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	outcome, err := provider.Poll(context.Background(), []PaymentOrder{{
		Id: 7, ProviderRef: "123", IdempotencyKey: "trusted", Amount: 100, Currency: "RUB",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(outcome.Paid) != 1 || outcome.Paid[0].OrderID != 7 {
		t.Fatalf("exact response was not accepted: %+v", outcome)
	}
}

func TestCryptoBotPollRejectsAmbiguousProviderReference(t *testing.T) {
	provider := &cryptoBotProvider{}
	_, err := provider.Poll(context.Background(), []PaymentOrder{
		{Id: 1, ProviderRef: "123"}, {Id: 2, ProviderRef: "123"},
	})
	if err == nil {
		t.Fatal("duplicate provider reference was accepted")
	}
}

func TestCryptoBotDeleteInvoiceTreatsMissingAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":400,"name":"INVOICE_NOT_FOUND"}}`))
	}))
	defer server.Close()
	provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
	if err := provider.DeleteInvoice(context.Background(), "123"); err != nil {
		t.Fatalf("missing invoice cancellation was not idempotent: %v", err)
	}
}

func TestCryptoBotCallRejectsMissingResultAndOversize(t *testing.T) {
	tests := []struct {
		name string
		body func(http.ResponseWriter)
	}{
		{name: "missing result", body: func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"ok":true}`))
		}},
		{name: "oversized", body: func(w http.ResponseWriter) {
			_, _ = w.Write(make([]byte, maxTelegramResponseBytes+1))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { test.body(w) }))
			defer server.Close()
			provider := &cryptoBotProvider{token: "secret", baseURL: server.URL, client: server.Client()}
			var out map[string]any
			if err := provider.call(context.Background(), http.MethodGet, "/api/getInvoices", nil, &out); err == nil {
				t.Fatal("invalid provider response was accepted")
			}
		})
	}
}
