package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) UpdateTaxiTypeByUser(ctx context.Context, userID string, taxiType service_dto.TaxiType) error {
	if !taxiType.IsValid() {
		return errorsx.ErrInvalidTaxiType
	}
	return s.repo.UpdateTaxiTypeByUserID(ctx, userID, string(taxiType))
}
