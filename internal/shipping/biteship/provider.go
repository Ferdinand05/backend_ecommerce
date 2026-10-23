package biteship

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ferdinand/ecommerce/internal/shipping"
)

type Provider struct {
	client *Client
}

func NewProvider(client *Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) SearchAreas(
	ctx context.Context,
	query string,
) ([]shipping.Area, error) {
	results, err := p.client.SearchAreas(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			shipping.ErrAreaProvider,
			err,
		)
	}

	areas := make([]shipping.Area, 0, len(results))

	for _, result := range results {
		if strings.TrimSpace(result.ID) == "" ||
			strings.TrimSpace(result.Name) == "" {
			continue
		}

		areas = append(areas, shipping.Area{
			ID:         result.ID,
			Name:       result.Name,
			PostalCode: result.PostalCode,
		})
	}

	return areas, nil
}

func (p *Provider) CalculateRates(
	ctx context.Context,
	req shipping.RatesProviderRequest,
) ([]shipping.ShippingRate, error) {

	items := make([]ratesAPIItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, ratesAPIItem{
			Name:     item.Name,
			Value:    item.Value,
			Weight:   item.Weight,
			Quantity: item.Quantity,
		})
	}

	pricing, err := p.client.CalculateRates(ctx, ratesAPIRequest{
		OriginAreaID:      req.OriginAreaID,
		DestinationAreaID: req.DestinationAreaID,
		Couriers:          req.Couriers,
		Items:             items,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", shipping.ErrAreaProvider, err)
	}

	rates := make([]shipping.ShippingRate, 0, len(pricing))
	for _, p := range pricing {
		rates = append(rates, shipping.ShippingRate{
			CourierCode:           p.CourierCode,
			CourierName:           p.CourierName,
			CourierServiceCode:    p.CourierServiceCode,
			CourierServiceName:    p.CourierServiceName,
			Description:           p.Description,
			Price:                 p.Price,
			ShipmentDurationRange: p.ShipmentDurationRange,
			ShipmentDurationUnit:  p.ShipmentDurationUnit,
		})
	}

	return rates, nil
}

var _ shipping.Provider = (*Provider)(nil)

// CreateOrder menerjemahkan request order publik ke struktur API
// Biteship. Error *APIError diteruskan apa adanya (errors.As tetap
// menemukannya) supaya pemanggil bisa membaca kode error provider.
func (p *Provider) CreateOrder(
	ctx context.Context,
	req CreateOrderRequest,
) (CreateOrderResult, error) {

	items := make([]orderAPIItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, orderAPIItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Weight:   item.Weight,
			Value:    item.Value,
		})
	}

	result, err := p.client.CreateOrder(ctx, orderAPIRequest{
		OriginContactName:       req.OriginContactName,
		OriginContactPhone:      req.OriginContactPhone,
		OriginAddress:           req.OriginAddress,
		OriginAreaID:            req.OriginAreaID,
		DestinationContactName:  req.DestinationContactName,
		DestinationContactPhone: req.DestinationContactPhone,
		DestinationAddress:      req.DestinationAddress,
		DestinationAreaID:       req.DestinationAreaID,
		DestinationPostalCode:   req.DestinationPostalCode,
		CourierCompany:          req.CourierCompany,
		CourierType:             req.CourierType,
		// Order dibuat sekarang; scheduled delivery belum dipakai.
		DeliveryType: "now",
		ReferenceID:  req.ReferenceID,
		Items:        items,
	})
	if err != nil {
		return CreateOrderResult{}, fmt.Errorf("biteship create order: %w", err)
	}

	return CreateOrderResult{
		ID:          result.ID,
		Status:      result.Status,
		WaybillID:   result.Courier.WaybillID,
		TrackingURL: trackingPageURL(result.Courier.WaybillID),
	}, nil
}

// GetTracking menerjemahkan riwayat resi Biteship menjadi event.
func (p *Provider) GetTracking(
	ctx context.Context,
	waybillID string,
) ([]TrackingEvent, error) {

	result, err := p.client.GetTracking(ctx, waybillID)
	if err != nil {
		return nil, fmt.Errorf("biteship tracking: %w", err)
	}

	events := make([]TrackingEvent, 0, len(result.History))
	for _, h := range result.History {
		event := TrackingEvent{
			Status:      h.Status,
			Description: h.Note,
		}
		if h.UpdatedAt != "" {
			parsed, perr := time.Parse(time.RFC3339, h.UpdatedAt)
			if perr == nil {
				event.OccurredAt = parsed
			}
		}
		events = append(events, event)
	}

	return events, nil
}

// trackingPageURL ponytail: halaman pelacakan publik Biteship; ganti
// bila link resmi berubah atau respons API menyediakan URL langsung.
func trackingPageURL(waybillID string) string {
	if waybillID == "" {
		return ""
	}
	return "https://biteship.com/track/" + waybillID
}
