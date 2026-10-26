package metrics

import (
	"context"
	"time"
)

type Service interface {
	Summary(ctx context.Context) (SummaryResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Summary(ctx context.Context) (SummaryResponse, error) {
	return s.repo.Summary(ctx, time.Now().AddDate(0, 0, -30))
}
