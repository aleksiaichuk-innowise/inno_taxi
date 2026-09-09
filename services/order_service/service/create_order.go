package service

import (
	"context"
	"fmt"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
	"github.com/google/uuid"
)

func (s OrderService) CreateOrder(ctx context.Context, input service_dto.CreateOrderInput) (service_dto.Order, error) {
	if !input.TaxiType.IsValid() {
		return service_dto.Order{}, errorsx.ErrInvalidTaxiType
	}
	if input.Start.IsZero() || input.Destination.IsZero() {
		return service_dto.Order{}, errorsx.ErrInvalidLocation
	}

	// Generated here, not left to the DB's DEFAULT uuidv7(), because the
	// wallet charge below needs an idempotency reference before the order
	// row exists - see the design doc's "Charge flow" for the full reasoning.
	orderID, err := uuid.NewV7()
	if err != nil {
		return service_dto.Order{}, fmt.Errorf("generate order id: %w", err)
	}
	price := priceForTaxiType(input.TaxiType)

	if err := s.wallet.Charge(ctx, input.UserID, price, orderID.String()); err != nil {
		return service_dto.Order{}, err
	}

	order, err := s.repo.CreateOrder(ctx, orderID.String(), price, input)
	if err != nil {
		// The rider was already charged but the order was never persisted -
		// refund so they aren't left paying for nothing. Best-effort: this is
		// the one step here that can't be made atomic with the write it's
		// compensating for (an HTTP call and a Postgres insert can't share a
		// transaction), same accepted trade-off as the Kafka publish below.
		if refundErr := s.wallet.Refund(ctx, input.UserID, price, orderID.String()); refundErr != nil {
			slog.ErrorContext(ctx, "refund after failed order insert failed", "order_id", orderID.String(), "error", refundErr)
		}
		return service_dto.Order{}, err
	}

	order = s.tryAssignDriver(ctx, order)

	// Publish is best-effort and post-commit: a failure here must not undo or
	// fail an already-persisted order (ARCHITECTURE.md Cons #6 — no outbox/
	// idempotency guarantee yet, acceptable until a real Kafka consumer exists).
	if err := s.gateway.PublishOrderCreated(ctx, order); err != nil {
		slog.ErrorContext(ctx, "publish order created event failed", "order_id", order.ID, "error", err)
	}

	return order, nil
}

// tryAssignDriver attempts to claim and assign a driver for a just-created
// order. Best-effort throughout: no driver being available right now is a
// normal outcome, not a reason to fail an order that's already been
// correctly charged - see the design doc's "order_service wiring" section.
// On any failure it returns the order unchanged (still created, unassigned).
func (s OrderService) tryAssignDriver(ctx context.Context, order service_dto.Order) service_dto.Order {
	driverID, ok, err := s.driver.ClaimAvailableDriver(ctx, string(order.TaxiType))
	if err != nil {
		slog.ErrorContext(ctx, "claim available driver failed", "order_id", order.ID, "error", err)
		return order
	}
	if !ok {
		return order
	}

	assigned, err := s.repo.AssignDriver(ctx, order.ID, driverID)
	if err != nil {
		slog.ErrorContext(ctx, "assign driver to order failed", "order_id", order.ID, "driver_id", driverID, "error", err)
		if releaseErr := s.driver.ReleaseDriver(ctx, driverID); releaseErr != nil {
			slog.ErrorContext(ctx, "release driver after failed assignment failed", "order_id", order.ID, "driver_id", driverID, "error", releaseErr)
		}
		return order
	}

	return assigned
}
