package service

import (
	"context"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/errorsx"
)

func (s *AnalyticService) GetOrderStats(ctx context.Context, from, to time.Time) (service_dto.OrderStats, error) {
	if from.After(to) {
		return service_dto.OrderStats{}, errorsx.ErrInvalidDateRange
	}
	return s.repo.GetOrderStats(ctx, from, to)
}
