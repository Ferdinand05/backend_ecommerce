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

type trackingAPIResponse struct {
	History []trackingAPIEvent `json:"history"`
}

type trackingAPIEvent struct {
	Note      string `json:"note"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
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

// GetTracking memanggil GET /v1/trackings/:waybill_id. Waybill yang
// belum tersedia (404) dianggap belum ada riwayat, bukan error.
func (c *Client) GetTracking(
	ctx context.Context,
	waybillID string,
) (trackingAPIResponse, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/v1/trackings/"+waybillID,
		nil,
	)
	if err != nil {
		return trackingAPIResponse{}, fmt.Errorf("create Biteship tracking request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return trackingAPIResponse{}, fmt.Errorf("send Biteship tracking request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return trackingAPIResponse{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return trackingAPIResponse{}, fmt.Errorf(
			"Biteship tracking returned HTTP status %d",
			resp.StatusCode,
		)
	}

	var result trackingAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return trackingAPIResponse{}, fmt.Errorf("decode Biteship tracking response: %w", err)
	}

	return result, nil
}
