package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

func (s *AnalyticService) IngestOrderEvent(ctx context.Context, evt service_dto.OrderEvent) error {
	return s.repo.InsertOrderEvent(ctx, evt)
}
