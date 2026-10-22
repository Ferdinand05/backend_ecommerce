package xendit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	sdk "github.com/xendit/xendit-go/v7"
	common "github.com/xendit/xendit-go/v7/common"
	invoice "github.com/xendit/xendit-go/v7/invoice"
)

var ErrInvalidInvoiceInput = errors.New("invalid invoice input")

// Status invoice dari Xendit; dipakai service untuk memutuskan
// apakah payment PENDING masih hidup, sudah terbayar, atau mati.
const (
	InvoiceStatusPending = "PENDING"
	InvoiceStatusPaid    = "PAID"
	InvoiceStatusSettled = "SETTLED"
	InvoiceStatusExpired = "EXPIRED"
)

type CreateInvoiceRequest struct {
	ExternalID      string
	Description     string
	Amount          float64
	Currency        string
	InvoiceDuration time.Duration
}

type InvoiceResult struct {
	ID          string
	ExternalID  string
	Status      string
	CheckoutURL string
	ExpiryDate  *time.Time
}

type Client interface {
	CreateInvoice(
		ctx context.Context,
		req CreateInvoiceRequest,
	) (InvoiceResult, error)

	// FindLatestInvoiceByExternalID mengembalikan invoice Xendit terbaru
	// untuk external_id tertentu.
	//
	// - invoice ditemukan  -> (hasil, nil)
	// - dipastikan TIDAK ada -> (nil, nil)
	// - Xendit gagal dihubungi (status tak pasti) -> (nil, error)
	FindLatestInvoiceByExternalID(
		ctx context.Context,
		externalID string,
	) (*InvoiceResult, error)
}

type client struct {
	sdk *sdk.APIClient
}

func NewClient(apiKey string) Client {
	return &client{
		sdk: sdk.NewClient(apiKey),
	}
}

func (c *client) CreateInvoice(
	ctx context.Context,
	req CreateInvoiceRequest,
) (InvoiceResult, error) {
	if req.ExternalID == "" {
		return InvoiceResult{}, fmt.Errorf(
			"%w: external ID is required",
			ErrInvalidInvoiceInput,
		)
	}

	if req.Amount <= 0 {
		return InvoiceResult{}, fmt.Errorf(
			"%w: amount must be greater than zero",
			ErrInvalidInvoiceInput,
		)
	}

	if req.Currency == "" {
		req.Currency = "IDR"
	}

	if req.InvoiceDuration <= 0 {
		return InvoiceResult{}, fmt.Errorf(
			"%w: invoice duration must be positive",
			ErrInvalidInvoiceInput,
		)
	}

	request := *invoice.NewCreateInvoiceRequest(
		req.ExternalID,
		req.Amount,
	)
	request.SetDescription(req.Description)
	request.SetCurrency(req.Currency)
	// Xendit minta invoice_duration berupa integer detik; sisa milidetik
	// dari time.Until() membuat nilainya pecahan dan API menolaknya.
	request.SetInvoiceDuration(float32(int(req.InvoiceDuration.Seconds())))

	result, _, err := c.sdk.InvoiceApi.
		CreateInvoice(ctx).
		CreateInvoiceRequest(request).
		Execute()
	if err != nil {
		// Log body asli: pesan generik "Invalid input data" tidak
		// menyebut field mana yang salah; rawResponse berisi detailnya.
		var sdkErr common.XenditSdkError
		if errors.As(err, &sdkErr) {
			slog.Error("xendit.create_invoice_rejected",
				"external_id", req.ExternalID,
				"amount", req.Amount,
				"currency", req.Currency,
				"invoice_duration_seconds", int(req.InvoiceDuration.Seconds()),
				"status", sdkErr.Status(),
				"error_code", sdkErr.ErrorCode(),
				"raw_response", sdkErr.RawResponse(),
			)
		}

		return InvoiceResult{}, fmt.Errorf(
			"creating Xendit invoice: %w",
			err,
		)
	}

	return toInvoiceResult(result), nil
}

func (c *client) FindLatestInvoiceByExternalID(
	ctx context.Context,
	externalID string,
) (*InvoiceResult, error) {
	if externalID == "" {
		return nil, fmt.Errorf(
			"%w: external ID is required",
			ErrInvalidInvoiceInput,
		)
	}

	invoices, _, err := c.sdk.InvoiceApi.
		GetInvoices(ctx).
		ExternalId(externalID).
		Limit(10).
		Execute()
	if err != nil {
		return nil, fmt.Errorf(
			"listing Xendit invoices by external ID: %w",
			err,
		)
	}

	if len(invoices) == 0 {
		return nil, nil
	}

	sort.Slice(invoices, func(i, j int) bool {
		return invoices[i].GetCreated().After(invoices[j].GetCreated())
	})

	latest := toInvoiceResult(&invoices[0])
	return &latest, nil
}

func toInvoiceResult(result *invoice.Invoice) InvoiceResult {
	var expiryDate *time.Time
	if d := result.GetExpiryDate(); !d.IsZero() {
		expiryDate = &d
	}

	return InvoiceResult{
		ID:          result.GetId(),
		ExternalID:  result.GetExternalId(),
		Status:      string(result.GetStatus()),
		CheckoutURL: result.GetInvoiceUrl(),
		ExpiryDate:  expiryDate,
	}
}
