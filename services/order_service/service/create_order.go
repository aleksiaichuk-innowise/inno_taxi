package service

import (
	"context"

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
	return s.repo.CreateOrder(ctx, input)
}
