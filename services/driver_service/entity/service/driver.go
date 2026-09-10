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

type Status string

const (
	StatusAvailable Status = "available"
	StatusOnTrip    Status = "on-trip"
	StatusOffline   Status = "offline"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusAvailable, StatusOnTrip, StatusOffline:
		return true
	default:
		return false
	}
}

type Driver struct {
	ID        string
	UserID    string
	TaxiType  TaxiType
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DriverProfile is a Driver enriched with rating stats read live from
// analytic_service - not persisted anywhere in driver_service itself, so it
// stays a separate type rather than fields on Driver.
type DriverProfile struct {
	Driver
	RatingAverage float64
	RatingCount   int64
}
