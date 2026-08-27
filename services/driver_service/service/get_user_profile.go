package service

import (
	"context"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

func (s DriverService) GetProfileByUser(ctx context.Context, id string) (service.Driver, error) {
	return s.repo.FindByUserID(ctx, id)
}
