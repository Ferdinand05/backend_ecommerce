package biteship

import (
	"context"
	"fmt"
	"strings"

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
