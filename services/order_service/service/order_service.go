package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, input service_dto.CreateOrderInput) (service_dto.Order, error)
}

type OrderGateway interface {
	PublishOrderCreated(ctx context.Context, order service_dto.Order) error
}

type OrderService struct {
	repo    OrderRepository
	gateway OrderGateway
}

func NewOrderService(repo OrderRepository, gateway OrderGateway) OrderService {
	return OrderService{repo: repo, gateway: gateway}
}
