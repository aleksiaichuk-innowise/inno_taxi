package http

type CreateDriverReq struct {
	UserID   string `json:"user_id" validate:"required"`
	TaxiType string `json:"type" validate:"required"`
}
