package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, id string, priceMinorUnits int64, input service_dto.CreateOrderInput) (service_dto.Order, error)
	GetOrderByID(ctx context.Context, id string) (service_dto.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, status service_dto.Status) (service_dto.Order, error)
}

type OrderGateway interface {
	PublishOrderCreated(ctx context.Context, order service_dto.Order) error
}

// WalletGateway is blocking by design: a charge must succeed before an
// order is considered created, and a refund must succeed before a
// cancellation is considered final - see the design doc's "Charge flow"/
// "Cancel flow" sections for why.
type WalletGateway interface {
	Charge(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
	Refund(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
}

type OrderService struct {
	repo    OrderRepository
	gateway OrderGateway
	wallet  WalletGateway
}

func NewOrderService(repo OrderRepository, gateway OrderGateway, wallet WalletGateway) OrderService {
	return OrderService{repo: repo, gateway: gateway, wallet: wallet}
}
