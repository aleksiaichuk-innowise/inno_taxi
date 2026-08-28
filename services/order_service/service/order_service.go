package service

import (
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/repository/pg_repo"
)

type OrderService struct {
	repo pg_repo.PgRepository
}

func NewOrderService(repo pg_repo.PgRepository) OrderService {
	return OrderService{repo: repo}
}
