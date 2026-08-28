package grpc

import (
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func taxiTypeFromProto(t order_service.TaxiType) service_dto.TaxiType {
	switch t {
	case order_service.TaxiType_TAXI_TYPE_ECONOMY:
		return service_dto.TaxiTypeEconomy
	case order_service.TaxiType_TAXI_TYPE_COMFORT:
		return service_dto.TaxiTypeComfort
	case order_service.TaxiType_TAXI_TYPE_BUSINESS:
		return service_dto.TaxiTypeBusiness
	default:
		return ""
	}
}

func locationFromProto(l *order_service.Location) service_dto.Location {
	return service_dto.Location{
		Lat: l.GetLat(),
		Lng: l.GetLong(),
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

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64OrZero(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
