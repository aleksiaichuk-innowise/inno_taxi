package service

import "time"

type DriverRatingEvent struct {
	OrderID  string
	DriverID string
	Rating   int32
	Comment  string
	RatedAt  time.Time
}

type DriverRatingStats struct {
	Average float64
	Count   int64
}
