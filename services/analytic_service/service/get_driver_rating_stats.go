package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (s *AnalyticService) GetDriverRatingStats(ctx context.Context, driverID string) (service_dto.DriverRatingStats, error) {
	return s.repo.GetDriverRatingStats(ctx, driverID)
}
