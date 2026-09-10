package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func TestStartTrip_NotFound(t *testing.T) {
	repo := &fakeOrderRepository{getErr: errorsx.ErrOrderNotFound}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestStartTrip_Unassigned(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound for an order with no assigned driver, got %v", err)
	}
}

func TestStartTrip_NotYourAssignment(t *testing.T) {
	driverID := "someone-else"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound (not distinguishing from not-yours), got %v", err)
	}
}

func TestStartTrip_WrongStatus(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotStartable) {
		t.Fatalf("expected ErrOrderNotStartable, got %v", err)
	}
}

func TestStartTrip_Succeeds(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	started := order
	started.Status = service_dto.StatusInProgress
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &started}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, &fakeSearchRepository{})

	got, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != service_dto.StatusInProgress {
		t.Fatalf("got status %q, want in_progress", got.Status)
	}
}

func TestStartTrip_IndexesOrderAfterStarting(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	started := order
	started.Status = service_dto.StatusInProgress
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &started}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusInProgress {
		t.Fatalf("expected the started order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestStartTrip_IndexFailureDoesNotFailStartTrip(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	started := order
	started.Status = service_dto.StatusInProgress
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &started}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.StartTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
