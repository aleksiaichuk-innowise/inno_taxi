package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

type DriverRepository interface {
	CreateDriver(ctx context.Context, dto *service_dto.CreateDriverInput) (service_dto.Driver, error)
	FindByUserID(ctx context.Context, id string) (service_dto.Driver, error)
	UpdateStatusByUserID(ctx context.Context, userID, status string) error
	UpdateTaxiTypeByUserID(ctx context.Context, userID, status string) error
	FindByStatus(ctx context.Context, status service_dto.Status) ([]service_dto.Driver, error)
}

type DriverService struct {
	repo DriverRepository
}

func NewDriverService(r DriverRepository) *DriverService {
	return &DriverService{
		repo: r,
	}
}
