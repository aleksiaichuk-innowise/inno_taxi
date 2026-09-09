package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

func TestClaimAvailableDriver_InvalidTaxiType(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	_, err := svc.ClaimAvailableDriver(context.Background(), "not-real")
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
}

func TestClaimAvailableDriver_NoneAvailable(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	_, err := svc.ClaimAvailableDriver(context.Background(), service_dto.TaxiTypeEconomy)
	if !errors.Is(err, errorsx.ErrNoAvailableDriver) {
		t.Fatalf("expected ErrNoAvailableDriver, got %v", err)
	}
}
