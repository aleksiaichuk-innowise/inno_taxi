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

	calledWith      service_dto.CreateOrderInput
	calledWithID    string
	calledWithPrice int64
	createCalled    bool

	getOrder     *service_dto.Order
	getErr       error
	updateOrder  *service_dto.Order
	updateErr    error
	updateCalled bool
}

func (f *fakeOrderRepository) CreateOrder(_ context.Context, id string, price int64, input service_dto.CreateOrderInput) (service_dto.Order, error) {
	f.createCalled = true
	f.calledWithID = id
	f.calledWithPrice = price
	f.calledWith = input
	if f.err != nil {
		return service_dto.Order{}, f.err
	}
	return *f.order, nil
}

func (f *fakeOrderRepository) GetOrderByID(_ context.Context, _ string) (service_dto.Order, error) {
	if f.getErr != nil {
		return service_dto.Order{}, f.getErr
	}
	return *f.getOrder, nil
}

func (f *fakeOrderRepository) UpdateOrderStatus(_ context.Context, _ string, _ service_dto.Status) (service_dto.Order, error) {
	f.updateCalled = true
	if f.updateErr != nil {
		return service_dto.Order{}, f.updateErr
	}
	return *f.updateOrder, nil
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

type fakeWalletGateway struct {
	chargeErr error
	refundErr error

	chargeCalled bool
	chargeArgs   [3]any // userID, amount, referenceID
	refundCalled bool
	refundArgs   [3]any
}

func (f *fakeWalletGateway) Charge(_ context.Context, userID string, amount int64, referenceID string) error {
	f.chargeCalled = true
	f.chargeArgs = [3]any{userID, amount, referenceID}
	return f.chargeErr
}

func (f *fakeWalletGateway) Refund(_ context.Context, userID string, amount int64, referenceID string) error {
	f.refundCalled = true
	f.refundArgs = [3]any{userID, amount, referenceID}
	return f.refundErr
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
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

	input := validInput()
	input.TaxiType = "not-a-real-type"

	_, err := svc.CreateOrder(context.Background(), input)
	if !errors.Is(err, errorsx.ErrInvalidTaxiType) {
		t.Fatalf("expected ErrInvalidTaxiType, got %v", err)
	}
	if wallet.chargeCalled {
		t.Fatal("wallet must not be charged when validation fails")
	}
}

func TestCreateOrder_InvalidLocation(t *testing.T) {
	repo := &fakeOrderRepository{}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

	input := validInput()
	input.Destination = service_dto.Location{}

	_, err := svc.CreateOrder(context.Background(), input)
	if !errors.Is(err, errorsx.ErrInvalidLocation) {
		t.Fatalf("expected ErrInvalidLocation, got %v", err)
	}
	if wallet.chargeCalled {
		t.Fatal("wallet must not be charged when validation fails")
	}
}

func TestCreateOrder_ChargeFailsInsufficientFunds(t *testing.T) {
	repo := &fakeOrderRepository{}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{chargeErr: errorsx.ErrInsufficientFunds}
	svc := NewOrderService(repo, gw, wallet)

	_, err := svc.CreateOrder(context.Background(), validInput())
	if !errors.Is(err, errorsx.ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("order must not be persisted when the charge fails")
	}
}

func TestCreateOrder_ChargesBeforePersisting(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
	if !wallet.chargeCalled {
		t.Fatal("expected wallet.Charge to be called")
	}
	if wallet.chargeArgs[0] != "user-1" || wallet.chargeArgs[1] != priceComfortMinorUnits {
		t.Fatalf("unexpected charge args: %+v", wallet.chargeArgs)
	}
	if !repo.createCalled {
		t.Fatal("expected the order to be persisted")
	}
	if repo.calledWithID == "" {
		t.Fatal("expected a generated order ID to be passed to the repository")
	}
	if repo.calledWithID != wallet.chargeArgs[2] {
		t.Fatalf("expected the charge reference ID to match the order ID passed to the repository: charge=%v repo=%v", wallet.chargeArgs[2], repo.calledWithID)
	}
	if repo.calledWithPrice != priceComfortMinorUnits {
		t.Fatalf("got price %d, want %d", repo.calledWithPrice, priceComfortMinorUnits)
	}
}

func TestCreateOrder_RefundsOnInsertFailure(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeOrderRepository{err: repoErr}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

	_, err := svc.CreateOrder(context.Background(), validInput())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
	if !wallet.refundCalled {
		t.Fatal("expected the wallet to be refunded after a failed insert")
	}
	if wallet.refundArgs[2] != repo.calledWithID {
		t.Fatalf("expected the refund reference ID to match the failed insert's order ID: refund=%v order=%v", wallet.refundArgs[2], repo.calledWithID)
	}
	if gw.called {
		t.Fatal("gateway must not be called when the order was never persisted")
	}
}

func TestCreateOrder_PublishesAfterPersisting(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

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
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, gw, wallet)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected CreateOrder to succeed despite publish failure, got %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}
