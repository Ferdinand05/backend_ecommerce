package payment

import (
	"context"
	"errors"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Jalur webhook yang tidak menyentuh transaksi DB (db nil aman).

func TestWebhookUnknownPaymentIsNoop(t *testing.T) {
	repo := newFakePaymentRepo()
	repo.latestErr = ErrPaymentNotFound
	repo.latest = modelsPaymentForWebhook()

	// FindByProviderInvoiceID & FindByExternalID sama-sama kosong.
	svc := NewService(nil, &fakeOrderRepo{}, repo, nil, &fakeXendit{})

	err := svc.HandleInvoiceCallback(context.Background(), InvoiceCallback{
		ID:         "inv-unknown",
		ExternalID: "PAY-unknown",
		Status:     "PAID",
	})
	if err != nil {
		t.Fatalf("err = %v, want nil (200 no-op)", err)
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
}

func TestWebhookExternalIDMismatchRejected(t *testing.T) {
	p := modelsPaymentForWebhook()
	p.ProviderInvoiceID = strPtr("inv-1")

	repo := newFakePaymentRepo()
	repo.payments[p.ID] = p
	repo.latest = p

	svc := NewService(nil, &fakeOrderRepo{}, repo, nil, &fakeXendit{})

	err := svc.HandleInvoiceCallback(context.Background(), InvoiceCallback{
		ID:         "inv-1",
		ExternalID: "PAY-BEDA",
		Status:     "PAID",
	})
	if !errors.Is(err, ErrInvalidPayment) {
		t.Fatalf("err = %v, want ErrInvalidPayment", err)
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
}

func TestWebhookAmountMismatchRejected(t *testing.T) {
	p := modelsPaymentForWebhook() // amount 120000
	p.ProviderInvoiceID = strPtr("inv-1")

	repo := newFakePaymentRepo()
	repo.payments[p.ID] = p

	svc := NewService(nil, &fakeOrderRepo{}, repo, nil, &fakeXendit{})

	err := svc.HandleInvoiceCallback(context.Background(), InvoiceCallback{
		ID:         "inv-1",
		ExternalID: p.ExternalID,
		Status:     "PAID",
		Amount:     999,
		Currency:   "IDR",
	})
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("err = %v, want ErrAmountMismatch", err)
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
}

func TestWebhookStatusIgnoredWithoutChange(t *testing.T) {
	p := modelsPaymentForWebhook()
	p.ProviderInvoiceID = strPtr("inv-1")

	repo := newFakePaymentRepo()
	repo.payments[p.ID] = p

	svc := NewService(nil, &fakeOrderRepo{}, repo, nil, &fakeXendit{})

	for _, status := range []string{"PENDING", "WEIRD_STATUS"} {
		if err := svc.HandleInvoiceCallback(context.Background(), InvoiceCallback{
			ID:         "inv-1",
			ExternalID: p.ExternalID,
			Status:     status,
		}); err != nil {
			t.Fatalf("status %q: err = %v, want nil", status, err)
		}
	}

	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status berubah = %+v, want tidak ada", repo.statusUpdates)
	}
}

func modelsPaymentForWebhook() models.Payment {
	return models.Payment{
		ID:         uuid.New(),
		OrderID:    uuid.New(),
		ExternalID: "PAY-1",
		Status:     string(PaymentStatusPending),
		Amount:     decimal.NewFromInt(120000),
		Currency:   "IDR",
	}
}

func strPtr(s string) *string {
	return &s
}
