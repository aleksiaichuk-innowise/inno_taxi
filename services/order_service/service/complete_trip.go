package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func (s OrderService) CompleteTrip(ctx context.Context, orderID, driverUserID string) (service_dto.Order, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return service_dto.Order{}, err
	}

	if order.DriverID == nil || *order.DriverID != driverUserID {
		return service_dto.Order{}, errorsx.ErrOrderNotFound
	}
	if order.Status != service_dto.StatusInProgress {
		return service_dto.Order{}, errorsx.ErrOrderNotCompletable
	}

	// Best-effort, same as CancelOrder's release-on-cancel: an operational
	// detail, not something completing the trip should fail over.
	if err := s.driver.ReleaseDriver(ctx, *order.DriverID); err != nil {
		slog.ErrorContext(ctx, "release driver on trip completion failed", "order_id", order.ID, "driver_id", *order.DriverID, "error", err)
	}

	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCompleted)
}
