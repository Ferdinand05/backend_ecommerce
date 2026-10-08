package order

import (
	"context"
	"errors"
	"testing"

	"ferdinand/ecommerce/internal/models"

	"github.com/google/uuid"
)

type fakeRepo struct {
	findByIDRes models.Order
	findByIDErr error

	findByIDForUpdateRes models.Order
	findByIDForUpdateErr error

	findAllByUserIDRes    OrderListResult
	findAllByUserIDUserID uuid.UUID
	findAllByUserIDPage   int
	findAllByUserIDSize   int

	findAllRes    OrderListResult
	findAllFilter OrderFilter

	updateStatusCalled bool
	updateStatusID     uuid.UUID
	updateStatusStatus string
	updateStatusErr    error

	historiesCreated []models.OrderStatusHistory
}

func (f *fakeRepo) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return f.findByIDRes, f.findByIDErr
}

func (f *fakeRepo) FindByIDForUpdate(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	return f.findByIDForUpdateRes, f.findByIDForUpdateErr
}

func (f *fakeRepo) FindAllByUserID(ctx context.Context, userID uuid.UUID, page int, pageSize int) (OrderListResult, error) {
	f.findAllByUserIDUserID = userID
	f.findAllByUserIDPage = page
	f.findAllByUserIDSize = pageSize
	return f.findAllByUserIDRes, nil
}

func (f *fakeRepo) FindAll(ctx context.Context, filter OrderFilter) (OrderListResult, error) {
	f.findAllFilter = filter
	return f.findAllRes, nil
}

func (f *fakeRepo) Create(ctx context.Context, order *models.Order) error {
	return nil
}

func (f *fakeRepo) CreateItem(ctx context.Context, item *models.OrderItem) error {
	return nil
}

func (f *fakeRepo) CreateAddress(ctx context.Context, address *models.OrderAddress) error {
	return nil
}

func (f *fakeRepo) CreateStatusHistory(ctx context.Context, history *models.OrderStatusHistory) error {
	f.historiesCreated = append(f.historiesCreated, *history)
	return nil
}

func (f *fakeRepo) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	f.updateStatusCalled = true
	f.updateStatusID = orderID
	f.updateStatusStatus = status
	return f.updateStatusErr
}

func TestFindMyOrdersPassesPagination(t *testing.T) {
	repo := &fakeRepo{findAllByUserIDRes: OrderListResult{Total: 5, Page: 2, PageSize: 20}}
	svc := NewService(repo, nil)

	userID := uuid.New()
	result, err := svc.FindMyOrders(context.Background(), userID, 2, 20)
	if err != nil {
		t.Fatalf("FindMyOrders() error = %v", err)
	}

	if repo.findAllByUserIDUserID != userID || repo.findAllByUserIDPage != 2 || repo.findAllByUserIDSize != 20 {
		t.Fatalf("repo args = (%v, %d, %d), want (%v, 2, 20)",
			repo.findAllByUserIDUserID, repo.findAllByUserIDPage, repo.findAllByUserIDSize, userID)
	}

	if result.Total != 5 {
		t.Fatalf("total = %d, want 5", result.Total)
	}
}

func TestFindMyOrder(t *testing.T) {
	userID := uuid.New()

	t.Run("owned", func(t *testing.T) {
		repo := &fakeRepo{findByIDRes: models.Order{ID: uuid.New(), UserID: userID, Status: "PENDING"}}
		svc := NewService(repo, nil)

		order, err := svc.FindMyOrder(context.Background(), userID, repo.findByIDRes.ID)
		if err != nil {
			t.Fatalf("FindMyOrder() error = %v", err)
		}

		if order.UserID != userID {
			t.Fatalf("order user = %v, want %v", order.UserID, userID)
		}
	})

	t.Run("other user order is 404", func(t *testing.T) {
		repo := &fakeRepo{findByIDRes: models.Order{ID: uuid.New(), UserID: uuid.New(), Status: "PENDING"}}
		svc := NewService(repo, nil)

		_, err := svc.FindMyOrder(context.Background(), userID, repo.findByIDRes.ID)
		if !errors.Is(err, ErrorOrderNotFound) {
			t.Fatalf("error = %v, want ErrorOrderNotFound", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &fakeRepo{findByIDErr: ErrorOrderNotFound}
		svc := NewService(repo, nil)

		_, err := svc.FindMyOrder(context.Background(), userID, uuid.New())
		if !errors.Is(err, ErrorOrderNotFound) {
			t.Fatalf("error = %v, want ErrorOrderNotFound", err)
		}
	})
}

func TestFindAllPassesFilter(t *testing.T) {
	repo := &fakeRepo{findAllRes: OrderListResult{Total: 1}}
	svc := NewService(repo, nil)

	status := OrderStatusPaid
	filter := OrderFilter{Status: &status, Page: 1, PageSize: 10}

	result, err := svc.FindAll(context.Background(), filter)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if repo.findAllFilter.Status != filter.Status || result.Total != 1 {
		t.Fatalf("filter = %+v, want status PAID", repo.findAllFilter)
	}
}

func TestUpdateStatusInvalidStatus(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil)

	_, err := svc.UpdateStatus(context.Background(), uuid.New(), OrderStatus("JUNK"), nil)
	if !errors.Is(err, ErrorInvalidOrderStatus) {
		t.Fatalf("error = %v, want ErrorInvalidOrderStatus", err)
	}

	if repo.updateStatusCalled {
		t.Fatal("repo must not be called for invalid status")
	}
}

// Transisi valid/invalid dan keberadaan history diuji di
// service_integration_test.go karena butuh transaksi database sungguhan.
