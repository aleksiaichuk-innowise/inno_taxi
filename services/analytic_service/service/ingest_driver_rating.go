package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (s *AnalyticService) IngestDriverRating(ctx context.Context, evt service_dto.DriverRatingEvent) error {
	return s.repo.InsertDriverRating(ctx, evt)
}
