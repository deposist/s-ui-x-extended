package paidsub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

type stubInvoiceProvider struct {
	invoice   *Invoice
	err       error
	deleted   []string
	created   int
	deleteErr error
}

func (p *stubInvoiceProvider) Kind() ProviderKind { return ProviderCryptoBot }
func (p *stubInvoiceProvider) Title(lang) string  { return "CryptoBot" }
func (p *stubInvoiceProvider) CreateInvoice(context.Context, *PaymentOrder, *Tariff, *model.Client) (*Invoice, error) {
	p.created++
	return p.invoice, p.err
}
func (p *stubInvoiceProvider) DeleteInvoice(_ context.Context, ref string) error {
	p.deleted = append(p.deleted, ref)
	return p.deleteErr
}

func TestCreateOrderDoesNotReturnCryptoBotURLWhenReferencePersistenceFails(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Id: 7}
	tariff := Tariff{Id: 9, Price: 100, Currency: "RUB", AddDays: 30}
	provider := &stubInvoiceProvider{invoice: &Invoice{Method: InvoiceURL, PayURL: "https://pay.example/invoice", ProviderRef: "123"}}
	ps := NewPaymentService()
	ps.providerOverride = provider
	ps.afterInvoiceCreated = func() {
		_ = db.Exec(`CREATE TRIGGER fail_provider_ref BEFORE UPDATE OF provider_ref ON payment_orders
			BEGIN SELECT RAISE(ABORT, 'database is locked'); END`).Error
	}

	order, invoice, err := ps.CreateOrder(context.Background(), &client, &tariff, ProviderCryptoBot, 42)
	if err == nil || invoice != nil {
		t.Fatalf("CreateOrder = order %+v, invoice %+v, err %v; want no payment URL", order, invoice, err)
	}
	if order == nil || len(provider.deleted) != 1 || provider.deleted[0] != "123" {
		t.Fatalf("created provider invoice was not cancelled: order=%+v deleted=%v", order, provider.deleted)
	}
	var stored PaymentOrder
	if db.First(&stored, order.Id).Error != nil || stored.Status != StatusFailed || stored.ProviderRef != "" || len(stored.ProviderPayload) != 0 {
		t.Fatalf("failed order was not persisted consistently: %+v", stored)
	}
}

func TestCreateOrderRetryWaitsForCryptoBotReconciliation(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Id: 7}
	tariff := Tariff{Id: 9, Price: 100, Currency: "RUB", AddDays: 30}
	existing := newPaymentOrder(&client, &tariff, ProviderCryptoBot, 42, tariff.Price, tariff.Currency, nowUnix(), 15)
	existing.Status = StatusRecoverable
	if err := db.Create(existing).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{invoice: &Invoice{PayURL: "https://pay.example/new", ProviderRef: "new"}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	order, invoice, err := ps.CreateOrder(context.Background(), &client, &tariff, ProviderCryptoBot, 42)
	if err == nil || order == nil || order.Id != existing.Id || invoice != nil {
		t.Fatalf("CreateOrder retry = order %+v, invoice %+v, err %v", order, invoice, err)
	}
	if provider.created != 0 {
		t.Fatalf("retry created %d additional provider invoices", provider.created)
	}
}

func TestCreateOrderKeepsRecoverableStateWhenCryptoBotCancelFails(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Id: 7}
	tariff := Tariff{Id: 9, Price: 100, Currency: "RUB", AddDays: 30}
	provider := &stubInvoiceProvider{
		invoice:   &Invoice{Method: InvoiceURL, PayURL: "https://pay.example/invoice", ProviderRef: "123"},
		deleteErr: errors.New("provider unavailable"),
	}
	ps := NewPaymentService()
	ps.providerOverride = provider
	ps.afterInvoiceCreated = func() {
		_ = db.Exec(`CREATE TRIGGER fail_provider_ref_recoverable BEFORE UPDATE OF provider_ref ON payment_orders
			BEGIN SELECT RAISE(ABORT, 'database is locked'); END`).Error
	}

	order, invoice, err := ps.CreateOrder(context.Background(), &client, &tariff, ProviderCryptoBot, 42)
	if err == nil || order == nil || invoice != nil {
		t.Fatalf("CreateOrder = order %+v, invoice %+v, err %v", order, invoice, err)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusRecoverable {
		t.Fatalf("uncancelled invoice status = %q; want recoverable", stored.Status)
	}
}

func TestCreateOrderPersistsCryptoBotReferenceBeforeReturningURL(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Id: 7}
	tariff := Tariff{Id: 9, Price: 100, Currency: "RUB", AddDays: 30}
	provider := &stubInvoiceProvider{invoice: &Invoice{Method: InvoiceURL, PayURL: "https://pay.example/invoice", ProviderRef: "123"}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	order, invoice, err := ps.CreateOrder(context.Background(), &client, &tariff, ProviderCryptoBot, 42)
	if err != nil || invoice == nil || invoice.PayURL == "" {
		t.Fatalf("CreateOrder = order %+v, invoice %+v, err %v", order, invoice, err)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ProviderRef != "123" || extractProviderRef(stored.ProviderPayload) != "123" || stored.ExternalURL != invoice.PayURL || stored.Status != StatusPending {
		t.Fatalf("stored invoice reference = %+v", stored)
	}
}

func TestCreateOrderReusesRecoverableCryptoBotOrder(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Id: 7}
	tariff := Tariff{Id: 9, Price: 100, Currency: "RUB", AddDays: 30}
	payload, _ := json.Marshal(map[string]string{"ref": "existing"})
	existing := newPaymentOrder(&client, &tariff, ProviderCryptoBot, 42, tariff.Price, tariff.Currency, nowUnix(), 15)
	existing.ProviderPayload = payload
	existing.ProviderRef = "existing"
	existing.ExternalURL = "https://pay.example/existing"
	if err := db.Create(existing).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{err: errors.New("must not create another invoice")}
	ps := NewPaymentService()
	ps.providerOverride = provider

	order, invoice, err := ps.CreateOrder(context.Background(), &client, &tariff, ProviderCryptoBot, 42)
	if err != nil || order.Id != existing.Id || invoice == nil || invoice.PayURL != existing.ExternalURL {
		t.Fatalf("CreateOrder retry = order %+v, invoice %+v, err %v", order, invoice, err)
	}
	var count int64
	if err := db.Model(&PaymentOrder{}).Where("provider = ? AND status = ?", ProviderCryptoBot, StatusPending).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("retry created %d pending invoices", count)
	}
}

func TestApplyPaidOrderPreservesCryptoBotProviderReference(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Enable: false, Name: "cryptobot-reference"}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	tariff := Tariff{Price: 100, Currency: "RUB", AddDays: 1}
	order := newPaymentOrder(&client, &tariff, ProviderCryptoBot, 42, tariff.Price, tariff.Currency, nowUnix(), 15)
	order.ProviderRef = "123"
	if err := db.Create(order).Error; err != nil {
		t.Fatal(err)
	}
	if applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "cryptobot:123", nil); err != nil || !applied {
		t.Fatalf("ApplyPaidOrder = %v, %v", applied, err)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ProviderRef != "123" {
		t.Fatalf("provider reference was erased: %+v", stored)
	}
}
