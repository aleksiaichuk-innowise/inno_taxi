package http

import "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/service"

type Handler struct {
	svc *service.WalletService
}

func NewHandler(svc *service.WalletService) *Handler {
	return &Handler{svc: svc}
}
