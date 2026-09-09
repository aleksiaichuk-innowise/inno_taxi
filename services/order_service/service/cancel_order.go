package service

import (
	"context"
	"fmt"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func (s OrderService) CancelOrder(ctx context.Context, orderID, userID string) (service_dto.Order, error) {
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return service_dto.Order{}, err
	}

	// Not found and "found but not yours" collapse to the same error - a
	// distinct response for the latter would confirm someone else's order
	// ID exists.
	if order.UserID != userID {
		return service_dto.Order{}, errorsx.ErrOrderNotFound
	}
	if order.Status != service_dto.StatusCreated && order.Status != service_dto.StatusDriverAssigned {
		return service_dto.Order{}, errorsx.ErrOrderNotCancellable
	}

	// Refund before flipping the status, mirroring CreateOrder's ordering:
	// the financial operation gates the state transition. Refund is
	// idempotent by reference_id (order.ID), so a retry after a transient
	// wallet failure here is always safe.
	if order.PriceMinorUnits != nil && *order.PriceMinorUnits > 0 {
		if err := s.wallet.Refund(ctx, order.UserID, *order.PriceMinorUnits, order.ID); err != nil {
			return service_dto.Order{}, fmt.Errorf("refund for cancelled order: %w", err)
		}
	}

	// Releasing the driver is best-effort, unlike the refund above: it's an
	// operational detail (the driver stays on-trip a bit longer than
	// necessary if this fails), not something the rider's cancellation
	// should fail over once their money's already back.
	if order.DriverID != nil {
		if err := s.driver.ReleaseDriver(ctx, *order.DriverID); err != nil {
			slog.ErrorContext(ctx, "release driver on order cancellation failed", "order_id", order.ID, "driver_id", *order.DriverID, "error", err)
		}
	}

	return s.repo.UpdateOrderStatus(ctx, orderID, service_dto.StatusCancelled)
}
