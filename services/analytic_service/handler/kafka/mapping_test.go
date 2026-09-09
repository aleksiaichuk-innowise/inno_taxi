package kafka

import (
	"testing"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func TestTaxiTypeString(t *testing.T) {
	cases := map[order_service.TaxiType]string{
		order_service.TaxiType_TAXI_TYPE_ECONOMY:     "economy",
		order_service.TaxiType_TAXI_TYPE_COMFORT:     "comfort",
		order_service.TaxiType_TAXI_TYPE_BUSINESS:    "business",
		order_service.TaxiType_TAXI_TYPE_UNSPECIFIED: "unspecified",
	}
	for in, want := range cases {
		if got := taxiTypeString(in); got != want {
			t.Errorf("taxiTypeString(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestStatusString(t *testing.T) {
	cases := map[order_service.Status]string{
		order_service.Status_STATUS_CREATED:         "created",
		order_service.Status_STATUS_DRIVER_ASSIGNED: "driver_assigned",
		order_service.Status_STATUS_IN_PROGRESS:     "in_progress",
		order_service.Status_STATUS_COMPLETED:       "completed",
		order_service.Status_STATUS_CANCELLED:       "cancelled",
		order_service.Status_STATUS_UNSPECIFIED:     "unspecified",
	}
	for in, want := range cases {
		if got := statusString(in); got != want {
			t.Errorf("statusString(%v) = %q, want %q", in, got, want)
		}
	}
}
