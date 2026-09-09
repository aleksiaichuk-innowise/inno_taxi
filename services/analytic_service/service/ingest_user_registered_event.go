package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (s *AnalyticService) IngestUserRegisteredEvent(ctx context.Context, evt service_dto.UserRegisteredEvent) error {
	return s.repo.InsertUserRegisteredEvent(ctx, evt)
}
