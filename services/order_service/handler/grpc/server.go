package grpc

import (
	"context"
	"errors"
	"time"

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
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &order_service.CreateOrderResponse{
		Order: &order_service.Order{
			Id:              res.ID,
			UserId:          res.UserID,
			DriverId:        stringOrEmpty(res.DriverID),
			TaxiType:        taxiTypeToProto(res.TaxiType),
			Start:           locationToProto(res.Start),
			Destination:     locationToProto(res.Destination),
			Status:          statusToProto(res.Status),
			PriceMinorUnits: int64OrZero(res.PriceMinorUnits),
			CreatedAt:       res.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}
