package shipping

import "context"

type Provider interface {
	SearchAreas(
		ctx context.Context,
		query string,
	) ([]Area, error)
}
