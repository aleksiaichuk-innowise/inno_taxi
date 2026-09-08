package gateway

import (
	"testing"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func TestOrderCreatedEventFromOrder(t *testing.T) {
	createdAt := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	order := service_dto.Order{
		ID:          "order-1",
		UserID:      "user-1",
		TaxiType:    service_dto.TaxiTypeBusiness,
		Start:       service_dto.Location{Lat: 1.5, Lng: 2.5},
		Destination: service_dto.Location{Lat: 3.5, Lng: 4.5},
		Status:      service_dto.StatusCreated,
		CreatedAt:   createdAt,
	}

	event := orderCreatedEventFromOrder(order)

	if event.GetOrderId() != order.ID {
		t.Errorf("OrderId = %q, want %q", event.GetOrderId(), order.ID)
	}
	if event.GetUserId() != order.UserID {
		t.Errorf("UserId = %q, want %q", event.GetUserId(), order.UserID)
	}
	if event.GetTaxiType() != order_service.TaxiType_TAXI_TYPE_BUSINESS {
		t.Errorf("TaxiType = %v, want TAXI_TYPE_BUSINESS", event.GetTaxiType())
	}
	if event.GetPickup().GetLat() != order.Start.Lat || event.GetPickup().GetLong() != order.Start.Lng {
		t.Errorf("Pickup = %+v, want %+v", event.GetPickup(), order.Start)
	}
	if event.GetDestination().GetLat() != order.Destination.Lat || event.GetDestination().GetLong() != order.Destination.Lng {
		t.Errorf("Destination = %+v, want %+v", event.GetDestination(), order.Destination)
	}
	if event.GetStatus() != order_service.Status_STATUS_CREATED {
		t.Errorf("Status = %v, want STATUS_CREATED", event.GetStatus())
	}
	if event.GetCreatedAt() != createdAt.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %q, want %q", event.GetCreatedAt(), createdAt.Format(time.RFC3339))
	}
}

func TestOrderCreatedEventFromOrder_UnknownEnumsMapToUnspecified(t *testing.T) {
	order := service_dto.Order{
		TaxiType: "not-a-real-type",
		Status:   "not-a-real-status",
	}

	event := orderCreatedEventFromOrder(order)

	if event.GetTaxiType() != order_service.TaxiType_TAXI_TYPE_UNSPECIFIED {
		t.Errorf("TaxiType = %v, want TAXI_TYPE_UNSPECIFIED", event.GetTaxiType())
	}
	if event.GetStatus() != order_service.Status_STATUS_UNSPECIFIED {
		t.Errorf("Status = %v, want STATUS_UNSPECIFIED", event.GetStatus())
	}
}
