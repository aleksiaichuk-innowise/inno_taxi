package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func TestCompleteTrip_NotFound(t *testing.T) {
	repo := &fakeOrderRepository{getErr: errorsx.ErrOrderNotFound}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestCompleteTrip_NotYourAssignment(t *testing.T) {
	driverID := "someone-else"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound (not distinguishing from not-yours), got %v", err)
	}
}

func TestCompleteTrip_WrongStatus(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, nil)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if !errors.Is(err, errorsx.ErrOrderNotCompletable) {
		t.Fatalf("expected ErrOrderNotCompletable, got %v", err)
	}
}

func TestCompleteTrip_ReleasesDriverThenUpdatesStatus(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	driver := &fakeDriverGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, driver, &fakeSearchRepository{})

	got, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != service_dto.StatusCompleted {
		t.Fatalf("got status %q, want completed", got.Status)
	}
	if !driver.releaseCalled || driver.releaseArg != "driver-1" {
		t.Fatalf("expected the driver to be released, got called=%v arg=%q", driver.releaseCalled, driver.releaseArg)
	}
}

func TestCompleteTrip_ReleaseFailureDoesNotBlockCompletion(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	driver := &fakeDriverGateway{releaseErr: errors.New("driver service unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, driver, &fakeSearchRepository{})

	got, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("expected completion to succeed despite a release failure, got %v", err)
	}
	if got.Status != service_dto.StatusCompleted {
		t.Fatalf("got status %q, want completed", got.Status)
	}
}

func TestCompleteTrip_IndexesOrderAfterCompleting(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusCompleted {
		t.Fatalf("expected the completed order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestCompleteTrip_IndexFailureDoesNotFailCompleteTrip(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress, DriverID: &driverID}
	completed := order
	completed.Status = service_dto.StatusCompleted
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &completed}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CompleteTrip(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
