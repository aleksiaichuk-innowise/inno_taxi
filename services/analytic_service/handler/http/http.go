package http

import "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"

type Handler struct {
	svc *service.AnalyticService
}

func NewHandler(svc *service.AnalyticService) *Handler {
	return &Handler{svc: svc}
}
