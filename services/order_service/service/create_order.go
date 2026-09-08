package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func (s OrderService) CreateOrder(ctx context.Context, input service_dto.CreateOrderInput) (service_dto.Order, error) {
	if !input.TaxiType.IsValid() {
		return service_dto.Order{}, errorsx.ErrInvalidTaxiType
	}
	if input.Start.IsZero() || input.Destination.IsZero() {
		return service_dto.Order{}, errorsx.ErrInvalidLocation
	}

	order, err := s.repo.CreateOrder(ctx, input)
	if err != nil {
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
