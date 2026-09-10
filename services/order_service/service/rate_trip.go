package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func (s OrderService) RateTrip(ctx context.Context, orderID, userID string, rating int32, comment *string) (service_dto.Order, error) {
	if rating < 1 || rating > 5 {
		return service_dto.Order{}, errorsx.ErrInvalidRating
	}

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return service_dto.Order{}, err
	}

	// Not found and "found but not yours" collapse to the same error, same
	// reasoning as every other order endpoint in this codebase.
	if order.UserID != userID {
		return service_dto.Order{}, errorsx.ErrOrderNotFound
	}
	if order.Status != service_dto.StatusCompleted {
		return service_dto.Order{}, errorsx.ErrOrderNotRatable
	}
	// Fast path only - the repository's conditional UPDATE is what actually
	// closes the race between two concurrent RateTrip calls for this order.
	if order.Rating != nil {
		return service_dto.Order{}, errorsx.ErrOrderAlreadyRated
	}

	rated, err := s.repo.RateOrder(ctx, orderID, rating, comment)
	if err != nil {
		return service_dto.Order{}, err
	}

	if err := s.search.IndexOrder(ctx, rated); err != nil {
		slog.ErrorContext(ctx, "index order in search failed", "order_id", rated.ID, "error", err)
	}

	// Best-effort and post-commit, same trade-off as order_created's publish.
	if err := s.gateway.PublishOrderRated(ctx, rated); err != nil {
		slog.ErrorContext(ctx, "publish order rated event failed", "order_id", rated.ID, "error", err)
	}

	return rated, nil
}
