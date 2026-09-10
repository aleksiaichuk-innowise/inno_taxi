package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// SearchOrders is an Analyst-only capability: no ownership scoping and no
// role check here, unlike ListOrders - role gating for this endpoint lives
// entirely in gateway_service (see the design doc's "Data flow: search").
func (s OrderService) SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error) {
	switch {
	case filter.Limit <= 0:
		filter.Limit = defaultListOrdersLimit
	case filter.Limit > maxListOrdersLimit:
		filter.Limit = maxListOrdersLimit
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.search.SearchOrders(ctx, filter)
}
