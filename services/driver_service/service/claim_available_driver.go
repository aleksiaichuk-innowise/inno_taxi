package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) ClaimAvailableDriver(ctx context.Context, taxiType service_dto.TaxiType) (service_dto.Driver, error) {
	if !taxiType.IsValid() {
		return service_dto.Driver{}, errorsx.ErrInvalidTaxiType
	}
	return s.repo.ClaimAvailableDriver(ctx, string(taxiType))
}
