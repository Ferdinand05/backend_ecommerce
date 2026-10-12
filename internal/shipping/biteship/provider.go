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

var _ shipping.Provider = (*Provider)(nil)
