package repository

import (
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

type Order struct {
	ID              string    `db:"id"`
	UserID          string    `db:"user_id"`
	DriverID        *string   `db:"driver_id"`
	TaxiType        string    `db:"taxi_type"`
	StartLat        float64   `db:"start_lat"`
	StartLng        float64   `db:"start_lng"`
	DestinationLat  float64   `db:"destination_lat"`
	DestinationLng  float64   `db:"destination_lng"`
	Status          string    `db:"status"`
	PriceMinorUnits *int64    `db:"price_minor_units"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

func (r Order) ToDomain() service.Order {
	return service.Order{
		ID:              r.ID,
		UserID:          r.UserID,
		DriverID:        r.DriverID,
		TaxiType:        service.TaxiType(r.TaxiType),
		Start:           service.Location{Lat: r.StartLat, Lng: r.StartLng},
		Destination:     service.Location{Lat: r.DestinationLat, Lng: r.DestinationLng},
		Status:          service.Status(r.Status),
		PriceMinorUnits: r.PriceMinorUnits,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}
