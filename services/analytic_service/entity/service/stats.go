package service

type OrderStats struct {
	TotalOrders      int64
	CountsByStatus   map[string]int64
	CountsByTaxiType map[string]int64
}

type DailyOrderCount struct {
	Date  string
	Count int64
}
