package service

import "time"

type TaxiType string

const (
	TaxiTypeEconomy  TaxiType = "economy"
	TaxiTypeComfort  TaxiType = "comfort"
	TaxiTypeBusiness TaxiType = "business"
)

func (t TaxiType) IsValid() bool {
	switch t {
	case TaxiTypeEconomy, TaxiTypeComfort, TaxiTypeBusiness:
		return true
	default:
		return false
	}
}

type Location struct {
	Lat float64
	Lng float64
}

func (l Location) IsZero() bool {
	return l.Lat == 0 && l.Lng == 0
}

type Status string

const (
	StatusCreated        Status = "created"
	StatusDriverAssigned Status = "driver_assigned"
	StatusInProgress     Status = "in_progress"
	StatusCompleted      Status = "completed"
	StatusCancelled      Status = "cancelled"
)

type Order struct {
	ID              string
	UserID          string
	DriverID        *string
	TaxiType        TaxiType
	Start           Location
	Destination     Location
	Status          Status
	PriceMinorUnits *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
