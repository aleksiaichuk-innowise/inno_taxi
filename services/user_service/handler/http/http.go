package http

import (
	"github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/service"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	userSvc  *service.UserService
	validate *validator.Validate
}

func NewHttpHandler(s *service.UserService, v *validator.Validate) *Handler {
	return &Handler{
		userSvc:  s,
		validate: v,
	}
}
