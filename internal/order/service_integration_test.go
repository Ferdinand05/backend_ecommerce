package order

import (
	"context"
	"errors"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOrderServiceUpdateStatusSuccess(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, db)

	user := seedUser(t, db)
	order := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	note := "payment confirmed"
	updated, err := svc.UpdateStatus(context.Background(), order.ID, OrderStatusPaid, &note)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	if updated.Status != "PAID" {
		t.Fatalf("status = %q, want PAID", updated.Status)
	}

	if len(updated.StatusHistories) != 1 {
		t.Fatalf("histories = %d, want 1", len(updated.StatusHistories))
	}

	if updated.StatusHistories[0].Status != "PAID" || updated.StatusHistories[0].Note == nil || *updated.StatusHistories[0].Note != note {
		t.Fatalf("history = %+v, want PAID with note", updated.StatusHistories[0])
	}

	// rantai transisi berikutnya tetap tercatat
	updated, err = svc.UpdateStatus(context.Background(), order.ID, OrderStatusProcessing, nil)
	if err != nil {
		t.Fatalf("UpdateStatus() second error = %v", err)
	}

	if updated.Status != "PROCESSING" || len(updated.StatusHistories) != 2 {
		t.Fatalf("status = %q, histories = %d, want PROCESSING with 2 histories",
			updated.Status, len(updated.StatusHistories))
	}
}

func TestOrderServiceUpdateStatusRejectsInvalidTransition(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo, db)

	user := seedUser(t, db)
	order := models.Order{
		ID:          uuid.New(),
		UserID:      user.ID,
		OrderNumber: "ORD-" + uuid.NewString(),
		Status:      "PENDING",
		Subtotal:    decimal.NewFromInt(100000),
		TotalAmount: decimal.NewFromInt(100000),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to seed order: %v", err)
	}

	_, err := svc.UpdateStatus(context.Background(), order.ID, OrderStatusShipped, nil)
	if !errors.Is(err, ErrorInvalidStatusTransition) {
		t.Fatalf("error = %v, want ErrorInvalidStatusTransition", err)
	}

	current, err := repo.FindByID(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if current.Status != "PENDING" {
		t.Fatalf("status = %q, want PENDING (tidak boleh berubah)", current.Status)
	}

	if len(current.StatusHistories) != 0 {
		t.Fatalf("histories = %d, want 0 (tidak boleh ada history)", len(current.StatusHistories))
	}
}

func TestOrderServiceUpdateStatusNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db), db)

	_, err := svc.UpdateStatus(context.Background(), uuid.New(), OrderStatusPaid, nil)
	if !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("error = %v, want ErrorOrderNotFound", err)
	}
}
