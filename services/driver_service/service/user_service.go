package service

import (
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/repository"
)

type DriverService struct {
	repo *repository.DriverRepository
}

func NewDriverService(r *repository.DriverRepository) *DriverService {
	return &DriverService{
		repo: r,
	}
}
