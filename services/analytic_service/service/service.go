package service

import (
	"context"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

type OrderEventRepository interface {
	InsertOrderEvent(ctx context.Context, evt service_dto.OrderEvent) error
	GetOrderStats(ctx context.Context, from, to time.Time) (service_dto.OrderStats, error)
	GetDailyOrderCounts(ctx context.Context, from, to time.Time) ([]service_dto.DailyOrderCount, error)
}

type AnalyticService struct {
	repo OrderEventRepository
}

func NewAnalyticService(repo OrderEventRepository) *AnalyticService {
	return &AnalyticService{repo: repo}
}
