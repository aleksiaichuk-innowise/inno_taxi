package gateway

import (
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func orderCreatedEventFromOrder(order service_dto.Order) *order_service.OrderCreatedEvent {
	return &order_service.OrderCreatedEvent{
		OrderId:     order.ID,
		UserId:      order.UserID,
		TaxiType:    taxiTypeToProto(order.TaxiType),
		Pickup:      locationToProto(order.Start),
		Destination: locationToProto(order.Destination),
		Status:      statusToProto(order.Status),
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
	}
}

func taxiTypeToProto(t service_dto.TaxiType) order_service.TaxiType {
	switch t {
	case service_dto.TaxiTypeEconomy:
		return order_service.TaxiType_TAXI_TYPE_ECONOMY
	case service_dto.TaxiTypeComfort:
		return order_service.TaxiType_TAXI_TYPE_COMFORT
	case service_dto.TaxiTypeBusiness:
		return order_service.TaxiType_TAXI_TYPE_BUSINESS
	default:
		return order_service.TaxiType_TAXI_TYPE_UNSPECIFIED
	}
}

func statusToProto(s service_dto.Status) order_service.Status {
	switch s {
	case service_dto.StatusCreated:
		return order_service.Status_STATUS_CREATED
	case service_dto.StatusDriverAssigned:
		return order_service.Status_STATUS_DRIVER_ASSIGNED
	case service_dto.StatusInProgress:
		return order_service.Status_STATUS_IN_PROGRESS
	case service_dto.StatusCompleted:
		return order_service.Status_STATUS_COMPLETED
	case service_dto.StatusCancelled:
		return order_service.Status_STATUS_CANCELLED
	default:
		return order_service.Status_STATUS_UNSPECIFIED
	}
}

func locationToProto(l service_dto.Location) *order_service.Location {
	return &order_service.Location{
		Lat:  l.Lat,
		Long: l.Lng,
	}
}
