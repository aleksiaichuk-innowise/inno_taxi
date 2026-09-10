package service

import (
	"context"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

// EventRepository backs ingestion for every event kind analytic_service
// consumes from Kafka (order_created, user_registered, order_rated) - one
// ClickHouse repository, one table per event kind.
type EventRepository interface {
	InsertOrderEvent(ctx context.Context, evt service_dto.OrderEvent) error
	GetOrderStats(ctx context.Context, from, to time.Time) (service_dto.OrderStats, error)
	GetDailyOrderCounts(ctx context.Context, from, to time.Time) ([]service_dto.DailyOrderCount, error)
	InsertUserRegisteredEvent(ctx context.Context, evt service_dto.UserRegisteredEvent) error
	InsertDriverRating(ctx context.Context, evt service_dto.DriverRatingEvent) error
	GetDriverRatingStats(ctx context.Context, driverID string) (service_dto.DriverRatingStats, error)
}

type AnalyticService struct {
	repo EventRepository
}

func NewAnalyticService(repo EventRepository) *AnalyticService {
	return &AnalyticService{repo: repo}
}
