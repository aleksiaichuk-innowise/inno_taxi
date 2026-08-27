package http

type CreateDriverReq struct {
	UserID   string `json:"user_id" validate:"required"`
	TaxiType string `json:"type" validate:"required"`
}

type UpdateDriverStatusReq struct {
	Status string `json:"status" validate:"required"`
}

type UpdateDriverTypeReq struct {
	Type string `json:"taxi_type" validate:"required"`
}
