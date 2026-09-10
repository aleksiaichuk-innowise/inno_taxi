package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

// GetOrder is viewable by either side of the trip - the rider who placed
// it, or the driver assigned to it - unlike every mutating endpoint here,
// which is one-sided by design. See the design doc for why.
func (s OrderService) GetOrder(ctx context.Context, orderID, callerID string) (service_dto.Order, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return service_dto.Order{}, err
	}

	isRider := order.UserID == callerID
	isDriver := order.DriverID != nil && *order.DriverID == callerID
	if !isRider && !isDriver {
		return service_dto.Order{}, errorsx.ErrOrderNotFound
	}

	return order, nil
}
