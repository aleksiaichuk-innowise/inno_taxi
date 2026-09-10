package grpc

import (
	"fmt"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func orderToProto(o service_dto.Order) *order_service.Order {
	return &order_service.Order{
		Id:              o.ID,
		UserId:          o.UserID,
		DriverId:        stringOrEmpty(o.DriverID),
		TaxiType:        taxiTypeToProto(o.TaxiType),
		Start:           locationToProto(o.Start),
		Destination:     locationToProto(o.Destination),
		Status:          statusToProto(o.Status),
		PriceMinorUnits: int64OrZero(o.PriceMinorUnits),
		CreatedAt:       o.CreatedAt.Format(time.RFC3339),
		Rating:          int32OrZero(o.Rating),
		Comment:         stringOrEmpty(o.Comment),
	}
}

func int32OrZero(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

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

func statusFromProto(s order_service.Status) service_dto.Status {
	switch s {
	case order_service.Status_STATUS_CREATED:
		return service_dto.StatusCreated
	case order_service.Status_STATUS_DRIVER_ASSIGNED:
		return service_dto.StatusDriverAssigned
	case order_service.Status_STATUS_IN_PROGRESS:
		return service_dto.StatusInProgress
	case order_service.Status_STATUS_COMPLETED:
		return service_dto.StatusCompleted
	case order_service.Status_STATUS_CANCELLED:
		return service_dto.StatusCancelled
	default:
		return ""
	}
}

// searchFilterFromProto converts the wire request to the domain filter,
// parsing the RFC3339 date-range strings here (not in the service) because
// a malformed string is a parse failure, not a business-rule validation -
// see the design doc's "Data flow: search" section.
func searchFilterFromProto(req *order_service.SearchOrdersRequest) (service_dto.OrderSearchFilter, error) {
	filter := service_dto.OrderSearchFilter{
		TaxiType:           taxiTypeFromProto(req.GetTaxiType()),
		Status:             statusFromProto(req.GetStatus()),
		MinPriceMinorUnits: req.GetMinPriceMinorUnits(),
		MaxPriceMinorUnits: req.GetMaxPriceMinorUnits(),
		Limit:              req.GetLimit(),
		Offset:             req.GetOffset(),
	}

	if s := req.GetCreatedAfter(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return service_dto.OrderSearchFilter{}, fmt.Errorf("invalid created_after: %w", err)
		}
		filter.CreatedAfter = t
	}
	if s := req.GetCreatedBefore(); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return service_dto.OrderSearchFilter{}, fmt.Errorf("invalid created_before: %w", err)
		}
		filter.CreatedBefore = t
	}

	return filter, nil
}
