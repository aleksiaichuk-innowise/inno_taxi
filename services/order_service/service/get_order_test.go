package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func TestGetOrder_NotFound(t *testing.T) {
	repo := &fakeOrderRepository{getErr: errorsx.ErrOrderNotFound}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.GetOrder(context.Background(), "order-1", "user-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestGetOrder_ViewableByRider(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	got, err := svc.GetOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "order-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestGetOrder_ViewableByAssignedDriver(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", DriverID: &driverID, Status: service_dto.StatusInProgress}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	got, err := svc.GetOrder(context.Background(), "order-1", "driver-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "order-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestGetOrder_NotYourOrder(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", DriverID: &driverID, Status: service_dto.StatusInProgress}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.GetOrder(context.Background(), "order-1", "someone-else")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound (not distinguishing from not-yours), got %v", err)
	}
}
