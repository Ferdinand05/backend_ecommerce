package shipment

import (
	"context"
	"errors"
	"testing"

	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/shipping/biteship"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// fakeRepo menggantikan Repository untuk menguji guard status tanpa DB.
type fakeRepo struct {
	shipment      models.Shipment
	exists        bool
	createErr     error
	updateApplied bool
	updateCalls   int
}

func (f *fakeRepo) Create(ctx context.Context, shipment *models.Shipment) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.shipment = *shipment
	f.exists = true
	return nil
}

func (f *fakeRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {
	if !f.exists || f.shipment.ID != id {
		return nil, ErrShipmentNotFound
	}
	fresh := f.shipment
	return &fresh, nil
}

func (f *fakeRepo) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*models.Shipment, error) {
	if !f.exists || f.shipment.OrderID != orderID {
		return nil, ErrShipmentNotFound
	}
	fresh := f.shipment
	return &fresh, nil
}

func (f *fakeRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeRepo) Update(ctx context.Context, shipment *models.Shipment) error {
	f.shipment = *shipment
	return nil
}

func (f *fakeRepo) UpdateIfStatus(ctx context.Context, shipment *models.Shipment, fromStatus ShipmentStatus) (bool, error) {
	f.updateCalls++
	if !f.updateApplied || f.shipment.Status != string(fromStatus) || f.shipment.ID != shipment.ID {
		// Simulasikan transaksi paralel yang mengubah status sebelum
		// service melakukan read-after-race.
		if f.shipment.ID == shipment.ID && f.shipment.Status == string(fromStatus) {
			f.shipment.Status = string(ShipmentStatusConfirmed)
		}
		return false, nil
	}
	f.shipment = *shipment
	return true, nil
}

func (f *fakeRepo) FindByTrackingNumber(ctx context.Context, trackingNumber string) (*models.Shipment, error) {
	return nil, ErrShipmentNotFound
}

func (f *fakeRepo) FindByProviderOrderID(ctx context.Context, providerOrderID string) (*models.Shipment, error) {
	return nil, ErrShipmentNotFound
}

func (f *fakeRepo) FindAll(ctx context.Context, filter ShipmentFilter) (ShipmentListResponse, error) {
	return ShipmentListResponse{}, nil
}

func (f *fakeRepo) FindByIDWithOrder(ctx context.Context, id uuid.UUID) (*models.Shipment, error) {
	return f.FindByID(ctx, id)
}

type fakeOrders struct {
	order models.Order
}

func (f *fakeOrders) FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error) {
	f.order.ID = orderID
	return f.order, nil
}

type fakeProvider struct {
	result              biteship.CreateOrderResult
	err                 error
	callCount           int
	orderDetail         biteship.OrderDetailResponse
	trackingByID        biteship.TrackingAPIResponse
	orderDetailCalls    []string
	trackingDetailCalls []string
	trackingCalls       []string
}

func (f *fakeProvider) CreateOrder(ctx context.Context, req biteship.CreateOrderRequest) (biteship.CreateOrderResult, error) {
	f.callCount++
	if f.err != nil {
		return biteship.CreateOrderResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeProvider) GetTracking(ctx context.Context, waybillID string) ([]biteship.TrackingEvent, error) {
	f.trackingCalls = append(f.trackingCalls, waybillID)
	return nil, nil
}

func (f *fakeProvider) GetTrackingDetail(ctx context.Context, trackingID string) (biteship.TrackingAPIResponse, error) {
	f.trackingDetailCalls = append(f.trackingDetailCalls, trackingID)
	return f.trackingByID, nil
}

func (f *fakeProvider) GetOrderDetail(ctx context.Context, orderID string) (biteship.OrderDetailResponse, error) {
	f.orderDetailCalls = append(f.orderDetailCalls, orderID)
	return f.orderDetail, nil
}

func paidOrder(t *testing.T) models.Order {
	t.Helper()

	return models.Order{
		ID:                 uuid.New(),
		UserID:             uuid.New(),
		Status:             "PAID",
		CourierCode:        "jne",
		CourierServiceCode: "reg",
		ShippingCost:       decimal.NewFromInt(20000),
		Address: &models.OrderAddress{
			RecipientName:  "Budi",
			Phone:          "0812",
			Address:        "Jl. Test 1",
			PostalCode:     "15142",
			BiteshipAreaID: ptrString("area-1"),
		},
		Items: []models.OrderItem{
			{ProductName: "Kaos", VariantName: "Merah", Quantity: 2, WeightGrams: 500},
		},
	}
}

func ptrString(s string) *string { return &s }

func newTestService(repo *fakeRepo, orders *fakeOrders, provider *fakeProvider) Service {
	return NewService(repo, orders, provider, Origin{Name: "Toko", Phone: "0896", Address: "Jl. Gudang", AreaID: "origin-1"})
}

func pendingShipment(orderID uuid.UUID) models.Shipment {
	company := "jne"
	courierType := "reg"
	return models.Shipment{
		ID:             uuid.New(),
		OrderID:        orderID,
		Provider:       "BITESHIP",
		CourierCompany: &company,
		CourierType:    &courierType,
		Status:         string(ShipmentStatusPending),
		ShippingCost:   decimal.NewFromInt(20000),
		Currency:       "IDR",
	}
}

func TestConfirmPendingSuccess(t *testing.T) {
	order := paidOrder(t)
	shp := pendingShipment(order.ID)
	repo := &fakeRepo{shipment: shp, exists: true, updateApplied: true}
	provider := &fakeProvider{result: biteship.CreateOrderResult{
		ID: "bite-1", Status: "confirmed", WaybillID: "JNE123", TrackingURL: "https://biteship.com/track/JNE123",
	}}
	svc := newTestService(repo, &fakeOrders{order: order}, provider)

	result, err := svc.ConfirmPending(context.Background(), shp.ID)
	if err != nil {
		t.Fatalf("ConfirmPending: %v", err)
	}
	if result.Status != string(ShipmentStatusConfirmed) {
		t.Fatalf("status = %q, want CONFIRMED", result.Status)
	}
	if result.TrackingNumber == nil || *result.TrackingNumber != "JNE123" {
		t.Fatalf("tracking number = %v, want JNE123", result.TrackingNumber)
	}
	if repo.shipment.ProviderOrderID == nil || *repo.shipment.ProviderOrderID != "bite-1" {
		t.Fatalf("provider order id not saved")
	}
	if repo.shipment.ShippedAt == nil {
		t.Fatalf("shipped_at not set")
	}
	if provider.callCount != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount)
	}
}

func TestConfirmPendingRejectsNonPending(t *testing.T) {
	order := paidOrder(t)
	shp := pendingShipment(order.ID)
	shp.Status = string(ShipmentStatusConfirmed)
	repo := &fakeRepo{shipment: shp, exists: true, updateApplied: true}
	provider := &fakeProvider{result: biteship.CreateOrderResult{ID: "bite-1"}}
	svc := newTestService(repo, &fakeOrders{order: order}, provider)

	if _, err := svc.ConfirmPending(context.Background(), shp.ID); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("err = %v, want ErrInvalidStatusTransition", err)
	}
	if provider.callCount != 0 {
		t.Fatalf("provider must not be called, got %d", provider.callCount)
	}
}

func TestConfirmPendingProviderDuplicate(t *testing.T) {
	order := paidOrder(t)
	shp := pendingShipment(order.ID)
	repo := &fakeRepo{shipment: shp, exists: true, updateApplied: true}
	provider := &fakeProvider{err: &biteship.APIError{StatusCode: 400, Code: 40002060, Message: "reference used"}}
	svc := newTestService(repo, &fakeOrders{order: order}, provider)

	_, err := svc.ConfirmPending(context.Background(), shp.ID)
	if !errors.Is(err, ErrProviderOrderExists) {
		t.Fatalf("err = %v, want ErrProviderOrderExists", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("repo must not be updated, got %d calls", repo.updateCalls)
	}
	if repo.shipment.Status != string(ShipmentStatusPending) {
		t.Fatalf("status = %q, want still PENDING", repo.shipment.Status)
	}
}

func TestConfirmPendingRaceLost(t *testing.T) {
	order := paidOrder(t)
	shp := pendingShipment(order.ID)
	// Proses lain menang: UpdateIfStatus return false, baris sudah CONFIRMED.
	repo := &fakeRepo{shipment: shp, exists: true, updateApplied: false}
	provider := &fakeProvider{result: biteship.CreateOrderResult{ID: "bite-1"}}
	svc := newTestService(repo, &fakeOrders{order: order}, provider)

	result, err := svc.ConfirmPending(context.Background(), shp.ID)
	if err != nil {
		t.Fatalf("ConfirmPending race lost: %v", err)
	}
	// Setelah race, service baca ulang baris yang sudah CONFIRMED.
	// Fake repo harus simulate ini: set status CONFIRMED sebelum FindByID ke-2.
	repo.shipment.Status = string(ShipmentStatusConfirmed)
	if result.Status != string(ShipmentStatusConfirmed) {
		t.Fatalf("status = %q, want CONFIRMED", result.Status)
	}
}

func TestRecoverPending(t *testing.T) {
	order := paidOrder(t)

	// Tanpa shipment: no-op.
	empty := &fakeRepo{}
	provider := &fakeProvider{result: biteship.CreateOrderResult{ID: "bite-1"}}
	svc := newTestService(empty, &fakeOrders{order: order}, provider)
	if err := svc.RecoverPending(context.Background(), order.ID); err != nil {
		t.Fatalf("RecoverPending no shipment: %v", err)
	}
	if provider.callCount != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.callCount)
	}

	// PENDING: dikonfirmasi.
	pending := &fakeRepo{shipment: pendingShipment(order.ID), exists: true, updateApplied: true}
	svc = newTestService(pending, &fakeOrders{order: order}, provider)
	if err := svc.RecoverPending(context.Background(), order.ID); err != nil {
		t.Fatalf("RecoverPending pending: %v", err)
	}
	if pending.shipment.Status != string(ShipmentStatusConfirmed) {
		t.Fatalf("status = %q, want CONFIRMED", pending.shipment.Status)
	}

	// Sudah CONFIRMED: no-op.
	confirmed := &fakeRepo{shipment: pendingShipment(order.ID), exists: true, updateApplied: true}
	confirmed.shipment.Status = string(ShipmentStatusConfirmed)
	svc = newTestService(confirmed, &fakeOrders{order: order}, provider)
	if err := svc.RecoverPending(context.Background(), order.ID); err != nil {
		t.Fatalf("RecoverPending confirmed: %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("provider calls = %d, want 1 (hanya kasus PENDING)", provider.callCount)
	}
}

func TestTrackResolvesTrackingIDViaOrder(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()
	providerOrderID := "biteship-order-123"
	trackingID := "OCrDQHoig3qLfBGCnAMN7ebf"

	shp := pendingShipment(orderID)
	shp.ProviderOrderID = &providerOrderID
	shp.Status = string(ShipmentStatusConfirmed)

	repo := &fakeRepo{shipment: shp, exists: true}
	provider := &fakeProvider{
		orderDetail: biteship.OrderDetailResponse{
			Courier: biteship.OrderDetailCourier{TrackingID: trackingID},
		},
	}
	order := &fakeOrders{order: models.Order{ID: orderID, UserID: userID}}
	svc := newTestService(repo, order, provider)

	resp, err := svc.Track(context.Background(), userID, shp.ID)
	if err != nil {
		t.Fatalf("Track error = %v", err)
	}
	if resp.ShipmentID != shp.ID.String() {
		t.Fatalf("shipment_id = %s, want %s", resp.ShipmentID, shp.ID)
	}
	if len(provider.orderDetailCalls) != 1 || provider.orderDetailCalls[0] != providerOrderID {
		t.Fatalf("GetOrderDetail calls = %v, want [%s]", provider.orderDetailCalls, providerOrderID)
	}
	if len(provider.trackingCalls) != 1 || provider.trackingCalls[0] != trackingID {
		t.Fatalf("GetTracking calls = %v, want [%s]", provider.trackingCalls, trackingID)
	}
}

func TestTrackReturnsEmptyEventsWhenProviderOrderIDMissing(t *testing.T) {
	userID := uuid.New()
	orderID := uuid.New()

	shp := pendingShipment(orderID)
	shp.ProviderOrderID = nil

	repo := &fakeRepo{shipment: shp, exists: true}
	provider := &fakeProvider{}
	order := &fakeOrders{order: models.Order{ID: orderID, UserID: userID}}
	svc := newTestService(repo, order, provider)

	resp, err := svc.Track(context.Background(), userID, shp.ID)
	if err != nil {
		t.Fatalf("Track error = %v", err)
	}
	if len(resp.Events) != 0 {
		t.Fatalf("events = %v, want empty for unconfirmed shipment", resp.Events)
	}
	if len(provider.orderDetailCalls) != 0 {
		t.Fatalf("GetOrderDetail called %d times, want 0", len(provider.orderDetailCalls))
	}
}

func TestCreatePendingDuplicateIsNoop(t *testing.T) {
	order := paidOrder(t)
	repo := &fakeRepo{createErr: ErrDuplicateShipment}
	svc := newTestService(repo, &fakeOrders{order: order}, &fakeProvider{})

	if err := svc.CreatePending(context.Background(), order); err != nil {
		t.Fatalf("CreatePending duplicate: %v", err)
	}
}

func TestCreatePendingRejectsIncompleteOrder(t *testing.T) {
	order := paidOrder(t)
	order.Address = nil
	repo := &fakeRepo{}
	svc := newTestService(repo, &fakeOrders{order: order}, &fakeProvider{})

	if err := svc.CreatePending(context.Background(), order); !errors.Is(err, ErrInvalidShipmentData) {
		t.Fatalf("err = %v, want ErrInvalidShipmentData", err)
	}
}

func TestCanTransitionChain(t *testing.T) {
	if !CanTransition(ShipmentStatusPending, ShipmentStatusConfirmed) {
		t.Fatal("PENDING harus boleh ke CONFIRMED")
	}
	if !CanTransition(ShipmentStatusConfirmed, ShipmentStatusDelivered) {
		t.Fatal("CONFIRMED harus boleh ke DELIVERED")
	}
	if CanTransition(ShipmentStatusDelivered, ShipmentStatusPending) {
		t.Fatal("DELIVERED tidak boleh berubah lagi")
	}
}
