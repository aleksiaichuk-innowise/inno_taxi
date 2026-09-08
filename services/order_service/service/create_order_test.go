package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

type fakeOrderRepository struct {
	order *service_dto.Order
	err   error

	calledWith service_dto.CreateOrderInput
}

func (f *fakeOrderRepository) CreateOrder(_ context.Context, input service_dto.CreateOrderInput) (service_dto.Order, error) {
	f.calledWith = input
	if f.err != nil {
		return service_dto.Order{}, f.err
	}
	return *f.order, nil
}

type fakeOrderGateway struct {
	err error

	called     bool
	calledWith service_dto.Order
}

func (f *fakeOrderGateway) PublishOrderCreated(_ context.Context, order service_dto.Order) error {
	f.called = true
	f.calledWith = order
	return f.err
}

func validInput() service_dto.CreateOrderInput {
	return service_dto.CreateOrderInput{
		UserID:      "user-1",
		TaxiType:    service_dto.TaxiTypeComfort,
		Start:       service_dto.Location{Lat: 1, Lng: 1},
		Destination: service_dto.Location{Lat: 2, Lng: 2},
	}
}

func TestCreateOrder_InvalidTaxiType(t *testing.T) {
	repo := &fakeOrderRepository{}
	gw := &fakeOrderGateway{}
	svc := NewOrderService(repo, gw)

	input := validInput()
	input.TaxiType = "not-a-real-type"

	_, err := svc.CreateOrder(context.Background(), input)
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
	if gw.called {
		t.Fatal("gateway must not be called when validation fails")
	}
}

func TestCreateOrder_InvalidLocation(t *testing.T) {
	repo := &fakeOrderRepository{}
	gw := &fakeOrderGateway{}
	svc := NewOrderService(repo, gw)

	input := validInput()
	input.Destination = service_dto.Location{}

	_, err := svc.CreateOrder(context.Background(), input)
	if !errors.Is(err, errorsx.ErrInvalidLocation) {
		t.Fatalf("expected ErrInvalidLocation, got %v", err)
	}
	if gw.called {
		t.Fatal("gateway must not be called when validation fails")
	}
}

func TestCreateOrder_RepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeOrderRepository{err: repoErr}
	gw := &fakeOrderGateway{}
	svc := NewOrderService(repo, gw)

	_, err := svc.CreateOrder(context.Background(), validInput())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
	if gw.called {
		t.Fatal("gateway must not be called when the order was never persisted")
	}
}

func TestCreateOrder_PublishesAfterPersisting(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{}
	svc := NewOrderService(repo, gw)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
	if !gw.called {
		t.Fatal("expected gateway.PublishOrderCreated to be called")
	}
	if gw.calledWith != created {
		t.Fatalf("expected gateway called with %+v, got %+v", created, gw.calledWith)
	}
}

func TestCreateOrder_PublishFailureDoesNotFailCreateOrder(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{err: errors.New("kafka unreachable")}
	svc := NewOrderService(repo, gw)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected CreateOrder to succeed despite publish failure, got %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}
