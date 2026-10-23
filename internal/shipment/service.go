package shipment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ferdinand/ecommerce/internal/models"
	"ferdinand/ecommerce/internal/shipping/biteship"

	"github.com/google/uuid"
)

// Service mengelola siklus shipment: pencatatan PENDING di transaksi
// webhook pembayaran, konfirmasi ke provider pengiriman, dan
// pelacakan.
type Service interface {
	// CreatePending mencatat shipment PENDING (snapshot kurir dari
	// order). Dipanggil di dalam transaksi webhook Xendit; replay
	// (shipment sudah ada) dianggap sukses.
	CreatePending(ctx context.Context, ord models.Order) error

	// ConfirmPending membuat order pengiriman di provider lalu
	// mengubah status menjadi CONFIRMED. Aman dipanggil berulang:
	// bila bukan PENDING lagi, tidak terjadi apa-apa.
	ConfirmPending(ctx context.Context, shipmentID uuid.UUID) (CreateShipmentResult, error)

	// RecoverPending mengkonfirmasi ulang shipment PENDING milik
	// sebuah order; dipakai jalur replay webhook dan tanpa shipment
	// maupun bukan PENDING bersifat no-op.
	RecoverPending(ctx context.Context, orderID uuid.UUID) error

	// Retry untuk admin: paksa konfirmasi ulang shipment PENDING.
	Retry(ctx context.Context, shipmentID uuid.UUID) (CreateShipmentResult, error)

	// FindMyShipmentByOrder: shipment milik user untuk order tertentu.
	FindMyShipmentByOrder(ctx context.Context, userID, orderID uuid.UUID) (ShipmentResponse, error)

	// FindMyShipment: shipment milik user berdasarkan ID shipment.
	FindMyShipment(ctx context.Context, userID, shipmentID uuid.UUID) (ShipmentResponse, error)

	// Track mengambil riwayat pelacakan live dari provider.
	Track(ctx context.Context, userID, shipmentID uuid.UUID) (ShipmentTrackingResponse, error)

	// ListShipments dipakai halaman admin; filter status opsional.
	ListShipments(ctx context.Context, filter ShipmentFilter) (ShipmentListResponse, error)

	// GetShipmentDetail memuat shipment dan ringkasan order terkait.
	GetShipmentDetail(ctx context.Context, shipmentID uuid.UUID) (ShipmentDetailResponse, error)
}

// ShipmentProvider adalah bagian provider pengiriman yang dipakai
// service; diimplementasi biteship.Provider.
type ShipmentProvider interface {
	CreateOrder(ctx context.Context, req biteship.CreateOrderRequest) (biteship.CreateOrderResult, error)
	GetTracking(ctx context.Context, waybillID string) ([]biteship.TrackingEvent, error)
}

// Origin adalah identitas pengirim (gudang/toko) untuk order provider.
type Origin struct {
	Name    string
	Phone   string
	Address string
	AreaID  string
}

// orderReader dibutuhkan untuk membaca order (alamat + item) saat
// konfirmasi; diimplementasi order.Repository.
type orderReader interface {
	FindByID(ctx context.Context, orderID uuid.UUID) (models.Order, error)
}

type service struct {
	repo     Repository
	orders   orderReader
	provider ShipmentProvider
	origin   Origin
}

func NewService(
	repo Repository,
	orders orderReader,
	provider ShipmentProvider,
	origin Origin,
) Service {
	return &service{
		repo:     repo,
		orders:   orders,
		provider: provider,
		origin:   origin,
	}
}

// CreatePending menyimpan shipment PENDING dengan snapshot kurir dan
// ongkir dari order. ID yang dibuat dipakai sebagai reference_id
// provider sehingga retry tidak menciptakan order ganda di provider.
func (s *service) CreatePending(ctx context.Context, ord models.Order) error {

	if ord.Address == nil || ord.CourierCode == "" || ord.CourierServiceCode == "" {
		return fmt.Errorf("%w: order %s lacks address or courier", ErrInvalidShipmentData, ord.ID)
	}

	courierCompany := ord.CourierCode
	courierType := ord.CourierServiceCode

	shipment := models.Shipment{
		ID:             uuid.New(),
		OrderID:        ord.ID,
		Provider:       "BITESHIP",
		CourierCompany: &courierCompany,
		CourierType:    &courierType,
		Status:         string(ShipmentStatusPending),
		ShippingCost:   ord.ShippingCost,
		Currency:       "IDR",
	}

	if err := s.repo.Create(ctx, &shipment); err != nil {
		if errors.Is(err, ErrDuplicateShipment) {
			// Replay webhook: shipment sudah tercatat.
			slog.InfoContext(ctx, "shipment.already_exists",
				"order_id", ord.ID)
			return nil
		}
		return err
	}

	slog.InfoContext(ctx, "shipment.created_pending",
		"shipment_id", shipment.ID,
		"order_id", ord.ID,
	)

	return nil
}

// ConfirmPending memanggil provider di luar transaksi DB lalu
// menyimpan hasilnya dengan guard status PENDING. Dua proses serentak
// aman: reference_id menahan duplikat di provider, UpdateIfStatus
// menahan overwrites di database.
func (s *service) ConfirmPending(
	ctx context.Context,
	shipmentID uuid.UUID,
) (CreateShipmentResult, error) {

	shp, err := s.repo.FindByIDForUpdate(ctx, shipmentID)
	if err != nil {
		return CreateShipmentResult{}, err
	}

	return s.confirm(ctx, *shp)
}

func (s *service) Retry(
	ctx context.Context,
	shipmentID uuid.UUID,
) (CreateShipmentResult, error) {

	result, err := s.ConfirmPending(ctx, shipmentID)
	if err != nil {
		return CreateShipmentResult{}, err
	}

	slog.InfoContext(ctx, "shipment.retried",
		"shipment_id", shipmentID,
	)

	return result, nil
}

// RecoverPending no-op untuk order tanpa shipment atau shipment yang
// sudah bukan PENDING.
func (s *service) RecoverPending(ctx context.Context, orderID uuid.UUID) error {

	shp, err := s.repo.FindByOrderID(ctx, orderID)
	if errors.Is(err, ErrShipmentNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if shp.Status != string(ShipmentStatusPending) {
		return nil
	}

	if _, err := s.confirm(ctx, *shp); err != nil {
		return err
	}

	slog.InfoContext(ctx, "shipment.recovered_pending",
		"shipment_id", shp.ID,
		"order_id", orderID,
	)

	return nil
}

func (s *service) confirm(ctx context.Context, shp models.Shipment) (CreateShipmentResult, error) {

	if shp.Status != string(ShipmentStatusPending) {
		return CreateShipmentResult{}, ErrInvalidStatusTransition
	}

	ord, err := s.orders.FindByID(ctx, shp.OrderID)
	if err != nil {
		return CreateShipmentResult{}, fmt.Errorf("finding order for shipment: %w", err)
	}

	req, err := buildCreateOrderRequest(s.origin, shp, ord)
	if err != nil {
		return CreateShipmentResult{}, err
	}

	result, err := s.provider.CreateOrder(ctx, req)
	if err != nil {
		var apiErr *biteship.APIError
		if errors.As(err, &apiErr) && apiErr.Code == 40002060 {
			// Provider sudah punya order dengan reference_id ini,
			// tetapi ID-nya tidak kita simpan (kegagalan sebelumnya).
			// Tidak bisa di-recover otomatis: API Biteship tidak
			// menyediakan lookup berdasarkan reference_id.
			slog.ErrorContext(ctx, "shipment.provider_order_exists",
				"shipment_id", shp.ID,
				"order_id", shp.OrderID,
			)
			return CreateShipmentResult{}, ErrProviderOrderExists
		}

		return CreateShipmentResult{}, fmt.Errorf("%w: %v", ErrProviderStateAmbiguous, err)
	}

	now := time.Now()
	confirmed := shp
	confirmed.Status = string(ShipmentStatusConfirmed)
	confirmed.ProviderOrderID = &result.ID
	if result.Status != "" {
		confirmed.ProviderStatus = &result.Status
	}
	if result.WaybillID != "" {
		confirmed.TrackingNumber = &result.WaybillID
	}
	if result.TrackingURL != "" {
		confirmed.TrackingURL = &result.TrackingURL
	}
	confirmed.ShippedAt = &now

	applied, err := s.repo.UpdateIfStatus(ctx, &confirmed, ShipmentStatusPending)
	if err != nil {
		return CreateShipmentResult{}, err
	}

	if !applied {
		// Proses lain menang race dan sudah mengkonfirmasi.
		fresh, ferr := s.repo.FindByID(ctx, shp.ID)
		if ferr != nil {
			return CreateShipmentResult{}, ferr
		}
		slog.InfoContext(ctx, "shipment.confirm_race_lost",
			"shipment_id", shp.ID,
			"status", fresh.Status,
		)
		return toCreateShipmentResult(*fresh), nil
	}

	slog.InfoContext(ctx, "shipment.confirmed",
		"shipment_id", shp.ID,
		"order_id", shp.OrderID,
		"provider_order_id", result.ID,
		"tracking_number", result.WaybillID,
	)

	return toCreateShipmentResult(confirmed), nil
}

func (s *service) FindMyShipmentByOrder(
	ctx context.Context,
	userID uuid.UUID,
	orderID uuid.UUID,
) (ShipmentResponse, error) {

	shp, err := s.repo.FindByOrderID(ctx, orderID)
	if err != nil {
		return ShipmentResponse{}, err
	}

	if err := s.ensureOwner(ctx, *shp, userID); err != nil {
		return ShipmentResponse{}, err
	}

	return toShipmentResponse(*shp), nil
}

func (s *service) FindMyShipment(
	ctx context.Context,
	userID uuid.UUID,
	shipmentID uuid.UUID,
) (ShipmentResponse, error) {

	shp, err := s.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return ShipmentResponse{}, err
	}

	if err := s.ensureOwner(ctx, *shp, userID); err != nil {
		return ShipmentResponse{}, err
	}

	return toShipmentResponse(*shp), nil
}

func (s *service) Track(
	ctx context.Context,
	userID uuid.UUID,
	shipmentID uuid.UUID,
) (ShipmentTrackingResponse, error) {

	shp, err := s.repo.FindByID(ctx, shipmentID)
	if err != nil {
		return ShipmentTrackingResponse{}, err
	}

	if err := s.ensureOwner(ctx, *shp, userID); err != nil {
		return ShipmentTrackingResponse{}, err
	}

	response := ShipmentTrackingResponse{
		ShipmentID:     shp.ID.String(),
		OrderID:        shp.OrderID.String(),
		Status:         shp.Status,
		TrackingNumber: shp.TrackingNumber,
		TrackingURL:    shp.TrackingURL,
		Events:         []ShipmentTrackingEvent{},
	}

	// Resi belum ada (kurir instant belum pickup) atau provider tidak
	// punya riwayat: kembalikan status lokal tanpa events.
	if shp.TrackingNumber == nil || *shp.TrackingNumber == "" {
		return response, nil
	}

	events, err := s.provider.GetTracking(ctx, *shp.TrackingNumber)
	if err != nil {
		return ShipmentTrackingResponse{}, fmt.Errorf(
			"fetching tracking for shipment %s: %w", shp.ID, err,
		)
	}

	for _, e := range events {
		response.Events = append(response.Events, ShipmentTrackingEvent{
			Status:      e.Status,
			Description: e.Description,
			OccurredAt:  e.OccurredAt,
		})
	}

	return response, nil
}

func (s *service) ListShipments(ctx context.Context, filter ShipmentFilter) (ShipmentListResponse, error) {
	return s.repo.FindAll(ctx, filter)
}

func (s *service) GetShipmentDetail(ctx context.Context, shipmentID uuid.UUID) (ShipmentDetailResponse, error) {

	shp, err := s.repo.FindByIDWithOrder(ctx, shipmentID)
	if err != nil {
		return ShipmentDetailResponse{}, err
	}

	return ShipmentDetailResponse{
		Shipment: toShipmentResponse(*shp),
		Order: OrderSummary{
			ID:          shp.Order.ID,
			UserID:      shp.Order.UserID,
			OrderNumber: shp.Order.OrderNumber,
			Status:      shp.Order.Status,
			TotalAmount: shp.Order.TotalAmount,
			CreatedAt:   shp.Order.CreatedAt,
		},
	}, nil
}

// ensureOwner memastikan shipment dimiliki order milik user; tidak
// cocok diperlakukan sebagai not found agar tidak membocorkan
// keberadaan shipment orang lain.
func (s *service) ensureOwner(ctx context.Context, shp models.Shipment, userID uuid.UUID) error {

	ord, err := s.orders.FindByID(ctx, shp.OrderID)
	if err != nil {
		return err
	}

	if ord.UserID != userID {
		return ErrShipmentNotFound
	}

	return nil
}

// buildCreateOrderRequest menyusun request provider dari snapshot
// order. Items memakai berat gram, satuan yang sama dengan Rates API.
func buildCreateOrderRequest(
	origin Origin,
	shp models.Shipment,
	ord models.Order,
) (biteship.CreateOrderRequest, error) {

	address := ord.Address
	if address == nil {
		return biteship.CreateOrderRequest{}, fmt.Errorf(
			"%w: order %s has no address snapshot",
			ErrInvalidShipmentData, ord.ID,
		)
	}

	company := pointerValue(shp.CourierCompany)
	courierType := pointerValue(shp.CourierType)
	if company == "" || courierType == "" {
		return biteship.CreateOrderRequest{}, fmt.Errorf(
			"%w: shipment %s has no courier snapshot",
			ErrInvalidShipmentData, shp.ID,
		)
	}

	items := make([]biteship.CreateOrderItem, 0, len(ord.Items))
	for _, item := range ord.Items {
		items = append(items, biteship.CreateOrderItem{
			Name:     item.ProductName + " - " + item.VariantName,
			Quantity: item.Quantity,
			Weight:   item.WeightGrams,
			Value:    item.Subtotal.IntPart(),
		})
	}

	return biteship.CreateOrderRequest{
		OriginContactName:       origin.Name,
		OriginContactPhone:      origin.Phone,
		OriginAddress:           origin.Address,
		OriginAreaID:            origin.AreaID,
		DestinationContactName:  address.RecipientName,
		DestinationContactPhone: address.Phone,
		DestinationAddress:      address.Address,
		DestinationAreaID:       pointerValue(address.BiteshipAreaID),
		DestinationPostalCode:   parsePostalCode(address.PostalCode),
		CourierCompany:          company,
		CourierType:             courierType,
		ReferenceID:             shp.ID.String(),
		Items:                   items,
	}, nil
}

func parsePostalCode(raw string) int {
	var code int
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0
		}
		code = code*10 + int(ch-'0')
	}
	return code
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func toCreateShipmentResult(shp models.Shipment) CreateShipmentResult {
	return CreateShipmentResult{
		ShipmentID:     shp.ID,
		OrderID:        shp.OrderID,
		Status:         shp.Status,
		TrackingNumber: shp.TrackingNumber,
		TrackingURL:    shp.TrackingURL,
	}
}
