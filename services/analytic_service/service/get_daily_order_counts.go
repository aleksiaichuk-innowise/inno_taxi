package service

import (
	"context"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/errorsx"
)

func (s *AnalyticService) GetDailyOrderCounts(ctx context.Context, from, to time.Time) ([]service_dto.DailyOrderCount, error) {
	if from.After(to) {
		return nil, errorsx.ErrInvalidDateRange
	}
	return s.repo.GetDailyOrderCounts(ctx, from, to)
}
