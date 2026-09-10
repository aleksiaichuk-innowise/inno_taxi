package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, id string, priceMinorUnits int64, input service_dto.CreateOrderInput) (service_dto.Order, error)
	GetOrderByID(ctx context.Context, id string) (service_dto.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, status service_dto.Status) (service_dto.Order, error)
	AssignDriver(ctx context.Context, id, driverID string) (service_dto.Order, error)
	RateOrder(ctx context.Context, id string, rating int32, comment *string) (service_dto.Order, error)
	ListOrdersByUser(ctx context.Context, userID string, limit, offset int32) ([]service_dto.Order, int64, error)
}

type OrderGateway interface {
	PublishOrderCreated(ctx context.Context, order service_dto.Order) error
	PublishOrderRated(ctx context.Context, order service_dto.Order) error
}

// WalletGateway is blocking by design: a charge must succeed before an
// order is considered created, and a refund must succeed before a
// cancellation is considered final - see the design doc's "Charge flow"/
// "Cancel flow" sections for why.
type WalletGateway interface {
	Charge(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
	Refund(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) error
}

// DriverGateway is best-effort, unlike WalletGateway: "no driver available"
// (or driver_service being briefly unreachable) is not a reason to fail an
// order that's already been correctly charged - see the driver-assignment
// design doc.
type DriverGateway interface {
	ClaimAvailableDriver(ctx context.Context, taxiType string) (userID string, ok bool, err error)
	ReleaseDriver(ctx context.Context, userID string) error
}

// SearchRepository indexes orders into Elasticsearch and serves search
// queries back for the Analyst-only SearchOrders endpoint. IndexOrder is
// best-effort at every call site (see search_orders.go and the six write
// points in create_order.go/cancel_order.go/start_trip.go/complete_trip.go/
// rate_trip.go) - a search-index write failing must never fail the order
// operation it's mirroring.
type SearchRepository interface {
	IndexOrder(ctx context.Context, order service_dto.Order) error
	SearchOrders(ctx context.Context, filter service_dto.OrderSearchFilter) ([]service_dto.Order, int64, error)
}

type OrderService struct {
	repo    OrderRepository
	gateway OrderGateway
	wallet  WalletGateway
	driver  DriverGateway
	search  SearchRepository
}

func NewOrderService(repo OrderRepository, gateway OrderGateway, wallet WalletGateway, driver DriverGateway, search SearchRepository) OrderService {
	return OrderService{repo: repo, gateway: gateway, wallet: wallet, driver: driver, search: search}
}
