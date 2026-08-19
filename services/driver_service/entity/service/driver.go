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

type CreateDriverInput struct {
	UserID   string
	TaxiType TaxiType
}
