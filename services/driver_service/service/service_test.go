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

func (f *fakeDriverRepository) ClaimAvailableDriver(_ context.Context, _ string) (service_dto.Driver, error) {
	if f.err != nil {
		return service_dto.Driver{}, f.err
	}
	return service_dto.Driver{}, errorsx.ErrNoAvailableDriver
}

func (f *fakeDriverRepository) FindByStatus(_ context.Context, _ service_dto.Status) ([]service_dto.Driver, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.drivers, nil
}

type fakeRatingsGateway struct {
	average float64
	count   int64
	err     error
}

func (f *fakeRatingsGateway) GetDriverRatingStats(_ context.Context, _ string) (float64, int64, error) {
	if f.err != nil {
		return 0, 0, f.err
	}
	return f.average, f.count, nil
}

func TestCreateDriver_InvalidTaxiType(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{}, nil)
	_, err := svc.CreateDriver(context.Background(), &service_dto.CreateDriverInput{UserID: "u1", TaxiType: "not-real"})
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
}

func TestCreateDriver_Valid(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{}, nil)
	d, err := svc.CreateDriver(context.Background(), &service_dto.CreateDriverInput{UserID: "u1", TaxiType: service_dto.TaxiTypeEconomy})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.UserID != "u1" || d.TaxiType != service_dto.TaxiTypeEconomy {
		t.Fatalf("unexpected driver: %+v", d)
	}
}

func TestGetDriversByStatus_InvalidStatus(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{}, nil)
	_, err := svc.GetDriversByStatus(context.Background(), "not-real")
	if !errors.Is(err, errorsx.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestUpdateStatusByUser_InvalidStatus(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo, nil)
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
	svc := NewDriverService(repo, nil)
	if err := svc.UpdateStatusByUser(context.Background(), "u1", service_dto.StatusAvailable); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateStatusCalledWith != [2]string{"u1", "available"} {
		t.Fatalf("unexpected repo call args: %v", repo.updateStatusCalledWith)
	}
}

func TestUpdateTaxiTypeByUser_InvalidType(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := NewDriverService(repo, nil)
	err := svc.UpdateTaxiTypeByUser(context.Background(), "u1", "not-real")
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
	if repo.updateTaxiTypeCalledWith != [2]string{} {
		t.Fatal("repository must not be called on invalid input")
	}
}

func TestGetProfileByUser_NotFound(t *testing.T) {
	svc := NewDriverService(&fakeDriverRepository{}, nil)
	_, err := svc.GetProfileByUser(context.Background(), "missing")
	if !errors.Is(err, errorsx.ErrDriverNotFound) {
		t.Fatalf("expected ErrDriverNotFound, got %v", err)
	}
}

func TestGetProfileByUser_IncludesRatingStats(t *testing.T) {
	driver := &service_dto.Driver{UserID: "u1", TaxiType: service_dto.TaxiTypeEconomy, Status: service_dto.StatusAvailable}
	repo := &fakeDriverRepository{driver: driver}
	ratings := &fakeRatingsGateway{average: 4.5, count: 12}
	svc := NewDriverService(repo, ratings)

	p, err := svc.GetProfileByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.RatingAverage != 4.5 || p.RatingCount != 12 {
		t.Fatalf("got rating (%v, %v), want (4.5, 12)", p.RatingAverage, p.RatingCount)
	}
	if p.UserID != "u1" {
		t.Fatalf("expected the underlying driver fields to still be populated, got %+v", p)
	}
}

func TestGetProfileByUser_RatingsGatewayErrorDoesNotFailProfile(t *testing.T) {
	driver := &service_dto.Driver{UserID: "u1", TaxiType: service_dto.TaxiTypeEconomy, Status: service_dto.StatusAvailable}
	repo := &fakeDriverRepository{driver: driver}
	ratings := &fakeRatingsGateway{err: errors.New("analytic service unreachable")}
	svc := NewDriverService(repo, ratings)

	p, err := svc.GetProfileByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.RatingAverage != 0 || p.RatingCount != 0 {
		t.Fatalf("expected zero-value rating stats when the gateway fails, got (%v, %v)", p.RatingAverage, p.RatingCount)
	}
	if p.UserID != "u1" {
		t.Fatalf("expected the profile to still be returned, got %+v", p)
	}
}
