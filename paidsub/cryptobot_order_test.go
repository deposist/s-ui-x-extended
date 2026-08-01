package paidsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/deposist/s-ui-x-extended/database/model"
)

type stubInvoiceProvider struct {
	invoice    *Invoice
	err        error
	deleted    []string
	created    int
	deleteErr  error
	deleteErrs map[string]error
	reconciled []ReconciledInvoice
	pollErr    error
	pollCalls  [][]uint
}

func (p *stubInvoiceProvider) Kind() ProviderKind { return ProviderCryptoBot }
func (p *stubInvoiceProvider) Title(lang) string  { return "CryptoBot" }
func (p *stubInvoiceProvider) CreateInvoice(context.Context, *PaymentOrder, *Tariff, *model.Client) (*Invoice, error) {
	p.created++
	return p.invoice, p.err
}
func (p *stubInvoiceProvider) DeleteInvoice(_ context.Context, ref string) error {
	p.deleted = append(p.deleted, ref)
	if err := p.deleteErrs[ref]; err != nil {
		return err
	}
	return p.deleteErr
}
func (p *stubInvoiceProvider) ReconcileInvoices(context.Context, []PaymentOrder) ([]ReconciledInvoice, error) {
	return p.reconciled, nil
}
func (p *stubInvoiceProvider) Poll(_ context.Context, orders []PaymentOrder) (PollOutcome, error) {
	ids := make([]uint, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.Id)
	}
	p.pollCalls = append(p.pollCalls, ids)
	return PollOutcome{}, p.pollErr
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

func TestReconcileCryptoBotCancelsLegacyInvoiceWithoutSnapshot(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-cancel", ProviderRef: "123", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{{
		OrderID: order.Id, ProviderRef: "123", PayURL: "https://pay.example/123", ProviderStatus: "active",
	}}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	if len(provider.deleted) != 1 || provider.deleted[0] != "123" {
		t.Fatalf("legacy invoice was not cancelled: %v", provider.deleted)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusFailed || stored.ExternalURL != "" {
		t.Fatalf("cancelled legacy order remained payable: %+v", stored)
	}
}

func TestReconcileCryptoBotKeepsPaidLegacyInvoiceRecoverable(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-paid-review", ProviderRef: "456", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{{
		OrderID: order.Id, ProviderRef: "456", Paid: true, ProviderStatus: "paid", ProviderChargeID: "cryptobot:456",
	}}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusRecoverable || stored.SnapshotVersion != 0 || len(provider.deleted) != 0 {
		t.Fatalf("paid legacy invoice was finalized without a snapshot: order=%+v deleted=%v", stored, provider.deleted)
	}
}

func TestReconcileCryptoBotCancelsStoredLegacyReferenceBeforeFinalizing(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-mismatched-ref", ProviderRef: "123", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{{
		OrderID: order.Id, ProviderRef: "123", PayURL: "https://pay.example/123", ProviderStatus: "active",
	}, {
		OrderID: order.Id, ProviderRef: "456", PayURL: "https://pay.example/456", ProviderStatus: "active",
	}}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	if len(provider.deleted) != 2 || provider.deleted[0] != "123" || provider.deleted[1] != "456" {
		t.Fatalf("not all possibly payable invoices were cancelled: %v", provider.deleted)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusFailed {
		t.Fatalf("fully cancelled legacy order status = %q; want failed", stored.Status)
	}
}

func TestReconcileCryptoBotRetriesPartialLegacyCancellation(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-partial-cancel", ProviderRef: "123", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{
		deleteErrs: map[string]error{"456": errors.New("provider timeout")},
		reconciled: []ReconciledInvoice{
			{OrderID: order.Id, ProviderRef: "123", ProviderStatus: "active"},
			{OrderID: order.Id, ProviderRef: "456", ProviderStatus: "active"},
		},
	}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusRecoverable {
		t.Fatalf("partially cancelled order status = %q; want recoverable", stored.Status)
	}
	provider.deleted = nil
	provider.deleteErrs = nil
	provider.reconciled = []ReconciledInvoice{{OrderID: order.Id, ProviderRef: "456", ProviderStatus: "active"}}

	reconcileCryptoBot(context.Background(), ps)

	if len(provider.deleted) != 1 || provider.deleted[0] != "456" {
		t.Fatalf("retry did not cancel the remaining invoice: %v", provider.deleted)
	}
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusFailed {
		t.Fatalf("fully recovered cancellation status = %q; want failed", stored.Status)
	}
}

func TestReconcileCryptoBotCancelsEveryUnpaidLegacyDuplicate(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-duplicates", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{
		{OrderID: order.Id, ProviderRef: "123", PayURL: "https://pay.example/123", ProviderStatus: "active"},
		{OrderID: order.Id, ProviderRef: "456", PayURL: "https://pay.example/456", ProviderStatus: "active"},
	}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	if len(provider.deleted) != 2 || provider.deleted[0] != "123" || provider.deleted[1] != "456" {
		t.Fatalf("not all duplicate invoices were cancelled: %v", provider.deleted)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusFailed {
		t.Fatalf("fully cancelled duplicate order status = %q; want failed", stored.Status)
	}
}

func TestReconcileCryptoBotKeepsLegacyOrderRecoverableWhenAnyDuplicateIsPaid(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-paid-duplicate", ProviderRef: "123", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{
		{OrderID: order.Id, ProviderRef: "123", PayURL: "https://pay.example/123", ProviderStatus: "active"},
		{OrderID: order.Id, ProviderRef: "456", Paid: true, ProviderStatus: "paid", ProviderChargeID: "cryptobot:456"},
	}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusRecoverable || stored.SnapshotVersion != 0 {
		t.Fatalf("paid legacy duplicate was finalized without a snapshot: %+v", stored)
	}
}

func TestApplyPaidOrderRejectsSnapshotlessLegacyOrderWithoutGrant(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Name: "legacy-no-grant", Enable: false, Expiry: 100, Volume: 200}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusPending, IdempotencyKey: "legacy-paid-without-snapshot", GrantedDays: 30,
		GrantedTrafficBytes: 4096, SnapshotVersion: 0,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	applied, _, err := NewPaymentService().ApplyPaidOrder(order.Id, "cryptobot:123", nil)
	if err == nil || applied {
		t.Fatalf("ApplyPaidOrder snapshotless legacy order = applied %v, err %v; want rejection", applied, err)
	}
	var storedClient model.Client
	if err := db.First(&storedClient, client.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedClient.Enable != client.Enable || storedClient.Expiry != client.Expiry || storedClient.Volume != client.Volume {
		t.Fatalf("snapshotless payment mutated client: before=%+v after=%+v", client, storedClient)
	}
	var storedOrder PaymentOrder
	if err := db.First(&storedOrder, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedOrder.Status != StatusPending || storedOrder.ProviderChargeID != "" {
		t.Fatalf("snapshotless payment mutated order: %+v", storedOrder)
	}
}

func TestReconcileCryptoBotCancellationRemainsFinalAfterRestart(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-restart", ProviderRef: "123",
		ProviderPayload: []byte(`{"ref":"123"}`), CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{{
		OrderID: order.Id, ProviderRef: "123", ProviderStatus: "active",
	}}}
	ps := NewPaymentService()
	ps.providerOverride = provider
	reconcileCryptoBot(context.Background(), ps)

	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusFailed || stored.SnapshotVersion != paymentOrderLegacyResolvedVersion ||
		stored.ProviderRef != "" || len(stored.ProviderPayload) != 0 {
		t.Fatalf("resolved legacy cancellation was resurrected after restart: %+v", stored)
	}
}

func TestReconcileCryptoBotSnapshotPrefersPaidDuplicate(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Name: "paid-duplicate", Enable: false}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "snapshot-paid-duplicate", GrantedDays: 1,
		SnapshotVersion: paymentOrderSnapshotVersion, CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{
		{OrderID: order.Id, ProviderRef: "123", ProviderStatus: "active"},
		{OrderID: order.Id, ProviderRef: "456", ProviderStatus: "paid", Paid: true, ProviderChargeID: "cryptobot:456"},
	}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusPaid || stored.ProviderRef != "456" || stored.ProviderChargeID != "cryptobot:456" {
		t.Fatalf("paid duplicate was not selected: %+v", stored)
	}
	if len(provider.deleted) != 1 || provider.deleted[0] != "123" {
		t.Fatalf("paid duplicate did not cancel active sibling: %v", provider.deleted)
	}
	var cancellations int64
	if err := db.Model(&InvoiceCancellation{}).Count(&cancellations).Error; err != nil {
		t.Fatal(err)
	}
	if cancellations != 0 {
		t.Fatalf("completed sibling cancellation remained queued: %d", cancellations)
	}
}

func TestReconcileCryptoBotLegacyPaidDuplicateCancelsActiveDuplicate(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "legacy-paid-active-duplicate", CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{reconciled: []ReconciledInvoice{
		{OrderID: order.Id, ProviderRef: "123", ProviderStatus: "active"},
		{OrderID: order.Id, ProviderRef: "456", ProviderStatus: "paid", Paid: true},
	}}
	ps := NewPaymentService()
	ps.providerOverride = provider

	reconcileCryptoBot(context.Background(), ps)

	if len(provider.deleted) != 1 || provider.deleted[0] != "123" {
		t.Fatalf("active duplicate beside paid legacy invoice was not cancelled: %v", provider.deleted)
	}
	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusRecoverable || stored.SnapshotVersion != 0 {
		t.Fatalf("paid legacy invoice left manual-review state: %+v", stored)
	}
}

func TestPaidDuplicateSiblingCancellationRetriesAfterRestart(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	client := model.Client{Name: "paid-duplicate-restart", Enable: false}
	if err := db.Create(&client).Error; err != nil {
		t.Fatal(err)
	}
	order := PaymentOrder{
		ClientId: client.Id, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
		Status: StatusRecoverable, IdempotencyKey: "snapshot-paid-duplicate-restart", GrantedDays: 1,
		SnapshotVersion: paymentOrderSnapshotVersion, CreatedAt: nowUnix(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{
		deleteErrs: map[string]error{"123": errors.New("provider timeout")},
		reconciled: []ReconciledInvoice{
			{OrderID: order.Id, ProviderRef: "123", ProviderStatus: "active"},
			{OrderID: order.Id, ProviderRef: "456", ProviderStatus: "paid", Paid: true, ProviderChargeID: "cryptobot:456"},
		},
	}
	ps := NewPaymentService()
	ps.providerOverride = provider
	reconcileCryptoBot(context.Background(), ps)

	var stored PaymentOrder
	if err := db.First(&stored, order.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != StatusPaid {
		t.Fatalf("paid duplicate was not granted before cancellation retry: %+v", stored)
	}
	var queued int64
	if err := db.Model(&InvoiceCancellation{}).Count(&queued).Error; err != nil || queued != 1 {
		t.Fatalf("failed sibling cancellation was not durable: count=%d err=%v", queued, err)
	}

	provider.deleted = nil
	provider.deleteErrs = nil
	provider.reconciled = nil
	reconcileCryptoBot(context.Background(), ps)
	if len(provider.deleted) != 1 || provider.deleted[0] != "123" {
		t.Fatalf("restart did not retry sibling cancellation: %v", provider.deleted)
	}
	if err := db.Model(&InvoiceCancellation{}).Count(&queued).Error; err != nil || queued != 0 {
		t.Fatalf("successful retry did not clear cancellation: count=%d err=%v", queued, err)
	}
}

func TestPollCryptoBotBatchesFairlyAndOutageDoesNotExpire(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	const orderCount = cryptoBotPollBatchSize*2 + 17
	orders := make([]PaymentOrder, 0, orderCount)
	for i := range orderCount {
		orders = append(orders, PaymentOrder{
			ClientId: 1, TariffId: 1, Provider: string(ProviderCryptoBot), Amount: 100, Currency: "RUB",
			Status: StatusPending, IdempotencyKey: fmt.Sprintf("poll-fair-%d", i),
			ProviderRef: fmt.Sprintf("%d", i+1), CreatedAt: nowUnix() - cryptoBotPollGraceSeconds - 10,
			SnapshotVersion: paymentOrderSnapshotVersion,
		})
	}
	if err := db.Create(&orders).Error; err != nil {
		t.Fatal(err)
	}
	provider := &stubInvoiceProvider{pollErr: errors.New("provider unavailable")}
	ps := NewPaymentService()
	ps.providerOverride = provider
	pollCryptoBot(context.Background(), ps)

	seen := make(map[uint]bool, orderCount)
	for _, batch := range provider.pollCalls {
		if len(batch) > cryptoBotPollBatchSize {
			t.Fatalf("provider poll request was not bounded: %d", len(batch))
		}
		for _, id := range batch {
			seen[id] = true
		}
	}
	if len(seen) != orderCount {
		t.Fatalf("provider outage starved pending orders: saw %d of %d", len(seen), orderCount)
	}
	var nonPending int64
	if err := db.Model(&PaymentOrder{}).Where("id IN ? AND status <> ?", orderIDs(orders), StatusPending).Count(&nonPending).Error; err != nil {
		t.Fatal(err)
	}
	if nonPending != 0 {
		t.Fatalf("provider outage expired or finalized %d potentially-paid orders", nonPending)
	}
}

func orderIDs(orders []PaymentOrder) []uint {
	ids := make([]uint, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.Id)
	}
	return ids
}
