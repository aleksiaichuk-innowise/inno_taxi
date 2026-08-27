package service

import (
	"context"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) GetDriversByStatus(ctx context.Context, status service.Status) ([]service.Driver, error) {
	if !status.IsValid() {
		return nil, errorsx.ErrInvalidStatus
	}
	return s.repo.FindByStatus(ctx, status)
}
