package service

import (
	"context"

	input "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) CreateDriver(ctx context.Context, dto *input.CreateDriverInput) (input.Driver, error) {
	if !dto.TaxiType.IsValid() {
		return input.Driver{}, errorsx.ErrInvalidTaxiType
	}

	return s.repo.CreateDriver(ctx, dto)
}
