package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
)

type fakeDriverRepository struct {
	driver *service_dto.Driver
	err    error

	drivers []service_dto.Driver

	updateStatusCalledWith   [2]string
	updateTaxiTypeCalledWith [2]string
}

func (f *fakeDriverRepository) CreateDriver(_ context.Context, dto *service_dto.CreateDriverInput) (service_dto.Driver, error) {
	if f.err != nil {
		return service_dto.Driver{}, f.err
	}
	return service_dto.Driver{UserID: dto.UserID, TaxiType: dto.TaxiType, Status: service_dto.StatusOffline}, nil
}

func (f *fakeDriverRepository) FindByUserID(_ context.Context, _ string) (service_dto.Driver, error) {
	if f.err != nil {
		return service_dto.Driver{}, f.err
	}
	if f.driver == nil {
		return service_dto.Driver{}, errorsx.ErrDriverNotFound
	}
	return *f.driver, nil
}

func (f *fakeDriverRepository) UpdateStatusByUserID(_ context.Context, userID, status string) error {
	f.updateStatusCalledWith = [2]string{userID, status}
	return f.err
}

func (f *fakeDriverRepository) UpdateTaxiTypeByUserID(_ context.Context, userID, taxiType string) error {
	f.updateTaxiTypeCalledWith = [2]string{userID, taxiType}
	return f.err
}

func (f *fakeDriverRepository) FindByStatus(_ context.Context, _ service_dto.Status) ([]service_dto.Driver, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.drivers, nil
}

func TestCreateDriver_InvalidTaxiType(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	_, err := svc.CreateDriver(context.Background(), &service_dto.CreateDriverInput{UserID: "u1", TaxiType: "not-real"})
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
}

func TestCreateDriver_Valid(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	d, err := svc.CreateDriver(context.Background(), &service_dto.CreateDriverInput{UserID: "u1", TaxiType: service_dto.TaxiTypeEconomy})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.UserID != "u1" || d.TaxiType != service_dto.TaxiTypeEconomy {
		t.Fatalf("unexpected driver: %+v", d)
	}
}

func TestGetDriversByStatus_InvalidStatus(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	_, err := svc.GetDriversByStatus(context.Background(), "not-real")
	if !errors.Is(err, errorsx.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestUpdateStatusByUser_InvalidStatus(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)
	err := svc.UpdateStatusByUser(context.Background(), "u1", "not-real")
	if !errors.Is(err, errorsx.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
	if repo.updateStatusCalledWith != [2]string{} {
		t.Fatal("repository must not be called on invalid input")
	}
}

func TestUpdateStatusByUser_Valid(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)
	if err := svc.UpdateStatusByUser(context.Background(), "u1", service_dto.StatusAvailable); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateStatusCalledWith != [2]string{"u1", "available"} {
		t.Fatalf("unexpected repo call args: %v", repo.updateStatusCalledWith)
	}
}

func TestUpdateTaxiTypeByUser_InvalidType(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo)
	err := svc.UpdateTaxiTypeByUser(context.Background(), "u1", "not-real")
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
	if repo.updateTaxiTypeCalledWith != [2]string{} {
		t.Fatal("repository must not be called on invalid input")
	}
}

func TestGetProfileByUser_NotFound(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{})
	_, err := svc.GetProfileByUser(context.Background(), "missing")
	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Fatalf("expected ErrDriverNotFound, got %v", err)
	}
}
