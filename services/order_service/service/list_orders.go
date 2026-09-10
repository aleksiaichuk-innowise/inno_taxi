package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

const (
	defaultListOrdersLimit = 20
	maxListOrdersLimit     = 100
)

// ListOrders needs no ownership check by construction - the repository
// query itself is scoped to the caller, so there's nothing to leak.
func (s OrderService) ListOrders(ctx context.Context, callerID string, limit, offset int32) ([]service_dto.Order, int64, error) {
	switch {
	case limit <= 0:
		limit = defaultListOrdersLimit
	case limit > maxListOrdersLimit:
		limit = maxListOrdersLimit
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListOrdersByUser(ctx, callerID, limit, offset)
}
