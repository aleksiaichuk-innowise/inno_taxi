package kafka

import "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"

func taxiTypeString(t order_service.TaxiType) string {
	switch t {
	case order_service.TaxiType_TAXI_TYPE_ECONOMY:
		return "economy"
	case order_service.TaxiType_TAXI_TYPE_COMFORT:
		return "comfort"
	case order_service.TaxiType_TAXI_TYPE_BUSINESS:
		return "business"
	default:
		return "unspecified"
	}
}

func statusString(s order_service.Status) string {
	switch s {
	case order_service.Status_STATUS_CREATED:
		return "created"
	case order_service.Status_STATUS_DRIVER_ASSIGNED:
		return "driver_assigned"
	case order_service.Status_STATUS_IN_PROGRESS:
		return "in_progress"
	case order_service.Status_STATUS_COMPLETED:
		return "completed"
	case order_service.Status_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}
