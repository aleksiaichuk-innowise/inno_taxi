package service

import "time"

type OrderEvent struct {
	OrderID   string
	UserID    string
	TaxiType  string
	Status    string
	CreatedAt time.Time
}
