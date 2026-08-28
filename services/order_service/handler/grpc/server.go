package grpc

import (
	"context"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

// var _ order_service.OrderServiceServer = (*OrderServer)(nil)
type OrderServer struct {
	order_service.UnimplementedOrderServiceServer
}

func NewOrderServer() *OrderServer {
	return &OrderServer{}
}

func (o OrderServer) CreateOrder(ctx context.Context, request *order_service.CreateOrderRequest) (*order_service.CreateOrderResponse, error) {
	//TODO implement me
	panic("implement me")
}
