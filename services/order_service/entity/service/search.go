package service

import "time"

type OrderSearchFilter struct {
	TaxiType           TaxiType
	Status             Status
	CreatedAfter       time.Time
	CreatedBefore      time.Time
	MinPriceMinorUnits int64
	MaxPriceMinorUnits int64
	Limit              int32
	Offset             int32
}
