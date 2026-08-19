package http

type DriverResp struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	TaxiType string `json:"taxi_type"`
	Status   string `json:"status"`
}
