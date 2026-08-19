package http

import (
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc      *service.DriverService
	validate *validator.Validate
}

func NewDriverHandler(s *service.DriverService, v *validator.Validate) *Handler {
	return &Handler{
		svc:      s,
		validate: v,
	}
}
