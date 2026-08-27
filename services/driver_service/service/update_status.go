package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func (s DriverService) UpdateStatusByUser(ctx context.Context, userID string, status service_dto.Status) error {
	if !status.IsValid() {
		return errorsx.ErrInvalidStatus
	}
	return s.repo.UpdateStatusByUserID(ctx, userID, string(status))
}
