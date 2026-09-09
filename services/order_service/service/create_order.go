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

	// Publish is best-effort and post-commit: a failure here must not undo or
	// fail an already-persisted order (ARCHITECTURE.md Cons #6 — no outbox/
	// idempotency guarantee yet, acceptable until a real Kafka consumer exists).
	if err := s.gateway.PublishOrderCreated(ctx, order); err != nil {
		slog.ErrorContext(ctx, "publish order created event failed", "order_id", order.ID, "error", err)
	}

	return order, nil
}
