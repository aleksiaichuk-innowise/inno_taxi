package http

import "github.com/aleksiaichuk-innowise/inno_taxi/services/auth_service/service"

type Handler struct {
	svc *service.AuthService
}

func NewHandler(svc *service.AuthService) *Handler {
	return &Handler{svc: svc}
}
