package order

import (
	"context"
	"errors"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
)

// stubMovementRepo mencatat jurnal stok yang dibuat saat cancel.
type stubMovementRepo struct {
	created []models.StockMovement
	err     error
}

func (s *stubMovementRepo) Create(ctx context.Context, movement models.StockMovement) error {
	if s.err != nil {
		return s.err
	}
	s.created = append(s.created, movement)
	return nil
}

func (s *stubMovementRepo) FindByID(ctx context.Context, id uuid.UUID) (models.StockMovement, error) {
	return models.StockMovement{}, errors.New("not implemented")
}

func (s *stubMovementRepo) FindAllByInventoryID(ctx context.Context, inventoryID uuid.UUID) ([]models.StockMovement, error) {
	return nil, errors.New("not implemented")
}

func (s *stubMovementRepo) FindAll(ctx context.Context, movementType *string, page, pageSize int) ([]models.StockMovement, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func cancelService(
	repo *fakeRepo,
	inv *stubInventoryRepo,
	movements *stubMovementRepo,
) *service {
	return NewService(repo, nil, nil, inv, nil, nil, movements)
}

func TestCancelOrderRejectsOtherUser(t *testing.T) {
	repo := &fakeRepo{findByIDForUpdateRes: models.Order{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Status: "PENDING",
	}}

	inv := newStubInventoryRepo()
	svc := cancelService(repo, inv, &stubMovementRepo{})

	_, err := svc.CancelOrder(context.Background(), uuid.New(), repo.findByIDForUpdateRes.ID)
	if !errors.Is(err, ErrorOrderNotFound) {
		t.Fatalf("err = %v, want ErrorOrderNotFound", err)
	}

	if len(inv.released) != 0 {
		t.Fatalf("released = %v, want no release", inv.released)
	}
}

func TestCancelOrderRejectsNonPending(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepo{findByIDForUpdateRes: models.Order{
		ID:     uuid.New(),
		UserID: userID,
		Status: "PAID",
	}}

	inv := newStubInventoryRepo()
	svc := cancelService(repo, inv, &stubMovementRepo{})

	_, err := svc.CancelOrder(context.Background(), userID, repo.findByIDForUpdateRes.ID)
	if !errors.Is(err, ErrorOrderNotCancellable) {
		t.Fatalf("err = %v, want ErrorOrderNotCancellable", err)
	}

	if len(inv.released) != 0 {
		t.Fatalf("released = %v, want no release", inv.released)
	}
}

func TestCancelOrderReleasesReservations(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()
	variantID := uuid.New()

	repo := &fakeRepo{
		findByIDForUpdateRes: models.Order{ID: orderID, UserID: userID, Status: "PENDING"},
		findByIDRes:          models.Order{ID: orderID, UserID: userID, Status: "CANCELLED"},
		findItemsRes: []models.OrderItem{
			{ID: uuid.New(), OrderID: orderID, ProductVariantID: variantID, Quantity: 3},
		},
	}

	inv := newStubInventoryRepo()
	movements := &stubMovementRepo{}
	svc := cancelService(repo, inv, movements)

	order, err := svc.CancelOrder(context.Background(), userID, orderID)
	if err != nil {
		t.Fatalf("CancelOrder() error = %v", err)
	}

	if order.Status != "CANCELLED" {
		t.Fatalf("status = %q, want CANCELLED", order.Status)
	}

	if inv.released[variantID] != 3 {
		t.Fatalf("released = %v, want 3 for variant %s", inv.released, variantID)
	}

	if !repo.updateStatusCalled || repo.updateStatusStatus != "CANCELLED" {
		t.Fatalf("update status = (%v, %q), want (true, CANCELLED)", repo.updateStatusCalled, repo.updateStatusStatus)
	}

	if len(repo.historiesCreated) != 1 || repo.historiesCreated[0].Status != "CANCELLED" {
		t.Fatalf("histories = %+v, want one CANCELLED", repo.historiesCreated)
	}

	if len(movements.created) != 1 {
		t.Fatalf("movements = %d, want 1", len(movements.created))
	}

	if movements.created[0].Type != "ORDER_CANCEL" || movements.created[0].Quantity != 3 {
		t.Fatalf("movement = %+v, want ORDER_CANCEL qty 3", movements.created[0])
	}
}
