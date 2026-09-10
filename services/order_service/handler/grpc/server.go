package grpc

import (
	"context"
	"errors"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/transport/grpc/interceptor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// var _ order_service.OrderServiceServer = (*OrderServer)(nil)
type OrderServer struct {
	order_service.UnimplementedOrderServiceServer
	svc service.OrderService
}

func NewOrderServer(svc service.OrderService) *OrderServer {
	return &OrderServer{
		svc: svc,
	}
}

func (o OrderServer) CreateOrder(ctx context.Context, req *order_service.CreateOrderRequest) (*order_service.CreateOrderResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetTaxiType() == order_service.TaxiType_TAXI_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "taxi type unspecified")
	}
	input := service_dto.CreateOrderInput{
		UserID:      userId,
		TaxiType:    taxiTypeFromProto(req.GetTaxiType()),
		Start:       locationFromProto(req.GetStart()),
		Destination: locationFromProto(req.GetDestination()),
	}
	res, err := o.svc.CreateOrder(ctx, input)
	if err != nil {
		if errors.Is(err, errorsx.ErrInvalidTaxiType) || errors.Is(err, errorsx.ErrInvalidLocation) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, errorsx.ErrInsufficientFunds) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &order_service.CreateOrderResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) CancelOrder(ctx context.Context, req *order_service.CancelOrderRequest) (*order_service.CancelOrderResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	res, err := o.svc.CancelOrder(ctx, req.GetOrderId(), userId)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrOrderNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, errorsx.ErrOrderNotCancellable):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &order_service.CancelOrderResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) StartTrip(ctx context.Context, req *order_service.StartTripRequest) (*order_service.StartTripResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	res, err := o.svc.StartTrip(ctx, req.GetOrderId(), userId)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrOrderNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, errorsx.ErrOrderNotStartable):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &order_service.StartTripResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) CompleteTrip(ctx context.Context, req *order_service.CompleteTripRequest) (*order_service.CompleteTripResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	res, err := o.svc.CompleteTrip(ctx, req.GetOrderId(), userId)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrOrderNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, errorsx.ErrOrderNotCompletable):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &order_service.CompleteTripResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) RateTrip(ctx context.Context, req *order_service.RateTripRequest) (*order_service.RateTripResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	var comment *string
	if req.GetComment() != "" {
		c := req.GetComment()
		comment = &c
	}

	res, err := o.svc.RateTrip(ctx, req.GetOrderId(), userId, req.GetRating(), comment)
	if err != nil {
		switch {
		case errors.Is(err, errorsx.ErrOrderNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, errorsx.ErrOrderNotRatable), errors.Is(err, errorsx.ErrOrderAlreadyRated), errors.Is(err, errorsx.ErrInvalidRating):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &order_service.RateTripResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) GetOrder(ctx context.Context, req *order_service.GetOrderRequest) (*order_service.GetOrderResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	res, err := o.svc.GetOrder(ctx, req.GetOrderId(), userId)
	if err != nil {
		if errors.Is(err, errorsx.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &order_service.GetOrderResponse{
		Order: orderToProto(res),
	}, nil
}

func (o OrderServer) ListOrders(ctx context.Context, req *order_service.ListOrdersRequest) (*order_service.ListOrdersResponse, error) {
	userId, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "missing user ID")
	}

	orders, total, err := o.svc.ListOrders(ctx, userId, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoOrders := make([]*order_service.Order, len(orders))
	for i, ord := range orders {
		protoOrders[i] = orderToProto(ord)
	}

	return &order_service.ListOrdersResponse{
		Orders: protoOrders,
		Total:  total,
	}, nil
}

// SearchOrders has no ownership scoping and no role check here, unlike
// every other endpoint in this file - it's an Analyst-only endpoint and
// role gating for it lives entirely in gateway_service (see the design
// doc's "Data flow: search"). It does not call
// interceptor.UserIDFromContext because nothing here needs the caller's
// identity.
func (o OrderServer) SearchOrders(ctx context.Context, req *order_service.SearchOrdersRequest) (*order_service.SearchOrdersResponse, error) {
	filter, err := searchFilterFromProto(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	orders, total, err := o.svc.SearchOrders(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoOrders := make([]*order_service.Order, len(orders))
	for i, ord := range orders {
		protoOrders[i] = orderToProto(ord)
	}

	return &order_service.SearchOrdersResponse{
		Orders: protoOrders,
		Total:  total,
	}, nil
}
