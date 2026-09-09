package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func (s OrderService) StartTrip(ctx context.Context, orderID, driverUserID string) (service_dto.Order, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return service_dto.Order{}, err
	}

	// Not found and "found but not assigned to you" collapse to the same
	// error, same reasoning as CancelOrder's ownership check.
	if order.DriverID == nil || *order.DriverID != driverUserID {
		return service_dto.Order{}, errorsx.ErrOrderNotFound
	}
	if order.Status != service_dto.StatusDriverAssigned {
		return service_dto.Order{}, errorsx.ErrOrderNotStartable
	}

	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusInProgress)
}
