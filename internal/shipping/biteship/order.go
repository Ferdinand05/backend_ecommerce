package biteship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIError membawa kode error Biteship (mis. 40002060 reference_id
// sudah terpakai) supaya pemanggil bisa membedakan kegagalan pasti
// dari hasil ambigu.
type APIError struct {
	StatusCode int
	Code       int64
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf(
		"biteship error %d (http %d): %s",
		e.Code, e.StatusCode, e.Message,
	)
}

// CreateOrderRequest adalah request pembuatan order pengiriman.
type CreateOrderRequest struct {
	OriginContactName  string
	OriginContactPhone string
	OriginAddress      string
	OriginAreaID       string

	DestinationContactName  string
	DestinationContactPhone string
	DestinationAddress      string
	// Optional: identitas area dari Maps API; kurir instant mewajibkan
	// koordinat, bukan area.
	DestinationAreaID     string
	DestinationPostalCode int

	CourierCompany string
	CourierType    string
	ReferenceID    string
	Items          []CreateOrderItem
}

type CreateOrderItem struct {
	Name     string
	Quantity int
	// Weight dalam gram, satuan yang sama dengan Rates API.
	Weight int
	Value  int64
}

// CreateOrderResult adalah bagian respons Biteship yang relevan.
type CreateOrderResult struct {
	ID          string
	Status      string
	WaybillID   string
	TrackingURL string
}

// TrackingEvent adalah satu entri riwayat pelacakan resi.
type TrackingEvent struct {
	Status      string
	Description string
	OccurredAt  time.Time
}

// --- struktur API Biteship ---

type orderAPIRequest struct {
	OriginContactName       string         `json:"origin_contact_name"`
	OriginContactPhone      string         `json:"origin_contact_phone"`
	OriginAddress           string         `json:"origin_address"`
	OriginAreaID            string         `json:"origin_area_id,omitempty"`
	DestinationContactName  string         `json:"destination_contact_name"`
	DestinationContactPhone string         `json:"destination_contact_phone"`
	DestinationAddress      string         `json:"destination_address"`
	DestinationAreaID       string         `json:"destination_area_id,omitempty"`
	DestinationPostalCode   int            `json:"destination_postal_code,omitempty"`
	CourierCompany          string         `json:"courier_company"`
	CourierType             string         `json:"courier_type"`
	DeliveryType            string         `json:"delivery_type"`
	ReferenceID             string         `json:"reference_id,omitempty"`
	Items                   []orderAPIItem `json:"items"`
}

type orderAPIItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Weight   int    `json:"weight"`
	Value    int64  `json:"value,omitempty"`
}

type orderAPIResponse struct {
	Success bool            `json:"success"`
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	Courier orderAPICourier `json:"courier"`
}

type orderAPICourier struct {
	WaybillID string `json:"waybill_id"`
	Company   string `json:"company"`
	Type      string `json:"type"`
}

type TrackingAPIResponse struct {
	Success     bool                `json:"success"`
	Message     string              `json:"message"`
	ID          string              `json:"id"`
	WaybillID   string              `json:"waybill_id"`
	OrderID     string              `json:"order_id"`
	Status      string              `json:"status"`
	Link        string              `json:"link"`
	Courier     trackingAPICourier  `json:"courier"`
	Origin      trackingAPILocation `json:"origin"`
	Destination trackingAPILocation `json:"destination"`
	History     []trackingAPIEvent  `json:"history"`
}

type trackingAPICourier struct {
	Company           string `json:"company"`
	Type              string `json:"type"`
	Name              string `json:"name"`
	Phone             string `json:"phone"`
	DriverName        string `json:"driver_name"`
	DriverPhone       string `json:"driver_phone"`
	DriverPhotoURL    string `json:"driver_photo_url"`
	DriverPlateNumber string `json:"driver_plate_number"`
}

type trackingAPILocation struct {
	ContactName string `json:"contact_name"`
	Address     string `json:"address"`
}

type trackingAPIEvent struct {
	Note        string `json:"note"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
	ServiceType string `json:"service_type"`
}

type biteshipErrorResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// CreateOrder memanggil POST /v1/orders.
func (c *Client) CreateOrder(
	ctx context.Context,
	reqBody orderAPIRequest,
) (orderAPIResponse, error) {

	var result orderAPIResponse

	body, err := json.Marshal(reqBody)
	if err != nil {
		return result, fmt.Errorf("marshal order request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/orders",
		bytes.NewReader(body),
	)
	if err != nil {
		return result, fmt.Errorf("create Biteship order request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return result, fmt.Errorf("send Biteship order request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, decodeOrderError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("decode Biteship order response: %w", err)
	}

	if !result.Success || result.ID == "" {
		return result, fmt.Errorf("Biteship order creation was unsuccessful")
	}

	return result, nil
}

// decodeOrderError membungkus respons gagal Biteship menjadi *APIError
// bila body bisa dibaca; kalau tidak, error generik.
func decodeOrderError(resp *http.Response) error {

	var errBody biteshipErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		return fmt.Errorf("Biteship order returned HTTP status %d", resp.StatusCode)
	}

	message := errBody.Message
	if message == "" {
		message = errBody.Error
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       errBody.Code,
		Message:    message,
	}
}

type OrderDetailResponse struct {
	Success     bool                   `json:"success"`
	ID          string                 `json:"id"`
	ShortID     string                 `json:"short_id"`
	Status      string                 `json:"status"`
	Price       int64                  `json:"price"`
	Courier     OrderDetailCourier     `json:"courier"`
	Origin      orderDetailAPILocation `json:"origin"`
	Destination orderDetailAPILocation `json:"destination"`
	Delivery    orderDetailAPIDelivery `json:"delivery"`
	Items       []orderDetailAPIItem   `json:"items"`
}

type OrderDetailCourier struct {
	TrackingID  string                   `json:"tracking_id"`
	WaybillID   string                   `json:"waybill_id"`
	Company     string                   `json:"company"`
	Type        string                   `json:"type"`
	Link        string                   `json:"link"`
	RoutingCode string                   `json:"routing_code"`
	History     []trackingAPIEvent       `json:"history"`
	Insurance   *orderDetailAPIInsurance `json:"insurance"`
}

type orderDetailAPIInsurance struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Note     string `json:"note"`
}

type orderDetailAPILocation struct {
	ContactName  string                   `json:"contact_name"`
	ContactPhone string                   `json:"contact_phone"`
	Address      string                   `json:"address"`
	PostalCode   int                      `json:"postal_code"`
	Coordinate   orderDetailAPICoordinate `json:"coordinate"`
}

type orderDetailAPICoordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type orderDetailAPIDelivery struct {
	Type     string `json:"type"`
	Datetime string `json:"datetime"`
	Distance int    `json:"distance"`
}

type orderDetailAPIItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Weight   int    `json:"weight"`
	Value    int64  `json:"value"`
}

// GetOrder memanggil GET /v1/orders/:id.
func (c *Client) GetOrder(
	ctx context.Context,
	orderID string,
) (OrderDetailResponse, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/v1/orders/"+orderID,
		nil,
	)
	if err != nil {
		return OrderDetailResponse{}, fmt.Errorf("create Biteship order request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return OrderDetailResponse{}, fmt.Errorf("send Biteship order request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return OrderDetailResponse{}, fmt.Errorf(
			"Biteship order returned HTTP status %d",
			resp.StatusCode,
		)
	}

	var result OrderDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return OrderDetailResponse{}, fmt.Errorf("decode Biteship order response: %w", err)
	}

	return result, nil
}

// GetTracking memanggil GET /v1/trackings/:waybill_id. Waybill yang
// belum tersedia (404) dianggap belum ada riwayat, bukan error.
func (c *Client) GetTracking(
	ctx context.Context,
	waybillID string,
) (TrackingAPIResponse, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/v1/trackings/"+waybillID,
		nil,
	)
	if err != nil {
		return TrackingAPIResponse{}, fmt.Errorf("create Biteship tracking request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return TrackingAPIResponse{}, fmt.Errorf("send Biteship tracking request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return TrackingAPIResponse{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return TrackingAPIResponse{}, fmt.Errorf(
			"Biteship tracking returned HTTP status %d",
			resp.StatusCode,
		)
	}

	var result TrackingAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TrackingAPIResponse{}, fmt.Errorf("decode Biteship tracking response: %w", err)
	}

	return result, nil
}
