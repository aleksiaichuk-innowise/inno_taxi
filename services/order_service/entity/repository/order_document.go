package repository

import (
	"time"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

// OrderDocument is the Elasticsearch-facing shape of an Order - json-tagged,
// mirroring how Order (above, in order.go) is db-tagged for Postgres.
type OrderDocument struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	DriverID        string    `json:"driver_id"`
	TaxiType        string    `json:"taxi_type"`
	StartLat        float64   `json:"start_lat"`
	StartLng        float64   `json:"start_lng"`
	DestinationLat  float64   `json:"destination_lat"`
	DestinationLng  float64   `json:"destination_lng"`
	Status          string    `json:"status"`
	PriceMinorUnits int64     `json:"price_minor_units"`
	Rating          int32     `json:"rating"`
	Comment         string    `json:"comment"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func OrderDocumentFromDomain(o service.Order) OrderDocument {
	doc := OrderDocument{
		ID:             o.ID,
		UserID:         o.UserID,
		TaxiType:       string(o.TaxiType),
		StartLat:       o.Start.Lat,
		StartLng:       o.Start.Lng,
		DestinationLat: o.Destination.Lat,
		DestinationLng: o.Destination.Lng,
		Status:         string(o.Status),
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
	if o.DriverID != nil {
		doc.DriverID = *o.DriverID
	}
	if o.PriceMinorUnits != nil {
		doc.PriceMinorUnits = *o.PriceMinorUnits
	}
	if o.Rating != nil {
		doc.Rating = *o.Rating
	}
	if o.Comment != nil {
		doc.Comment = *o.Comment
	}
	return doc
}

func (d OrderDocument) ToDomain() service.Order {
	o := service.Order{
		ID:          d.ID,
		UserID:      d.UserID,
		TaxiType:    service.TaxiType(d.TaxiType),
		Start:       service.Location{Lat: d.StartLat, Lng: d.StartLng},
		Destination: service.Location{Lat: d.DestinationLat, Lng: d.DestinationLng},
		Status:      service.Status(d.Status),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if d.DriverID != "" {
		driverID := d.DriverID
		o.DriverID = &driverID
	}
	if d.PriceMinorUnits != 0 {
		price := d.PriceMinorUnits
		o.PriceMinorUnits = &price
	}
	if d.Rating != 0 {
		rating := d.Rating
		o.Rating = &rating
	}
	if d.Comment != "" {
		comment := d.Comment
		o.Comment = &comment
	}
	return o
}
