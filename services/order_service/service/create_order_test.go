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

	assignedOrder      *service_dto.Order
	assignDriverErr    error
	assignDriverCalled bool
	assignDriverArg    string

	ratedOrder      *service_dto.Order
	rateOrderErr    error
	rateOrderCalled bool

	listOrders       []service_dto.Order
	listTotal        int64
	listErr          error
	listCalled       bool
	listCalledUserID string
	listCalledLimit  int32
	listCalledOffset int32
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

func (f *fakeOrderRepository) AssignDriver(_ context.Context, _, driverID string) (service_dto.Order, error) {
	f.assignDriverCalled = true
	f.assignDriverArg = driverID
	if f.assignDriverErr != nil {
		return service_dto.Order{}, f.assignDriverErr
	}
	return *f.assignedOrder, nil
}

func (f *fakeOrderRepository) RateOrder(_ context.Context, _ string, rating int32, comment *string) (service_dto.Order, error) {
	f.rateOrderCalled = true
	if f.rateOrderErr != nil {
		return service_dto.Order{}, f.rateOrderErr
	}
	if f.ratedOrder != nil {
		return *f.ratedOrder, nil
	}
	return service_dto.Order{Rating: &rating, Comment: comment}, nil
}

func (f *fakeOrderRepository) ListOrdersByUser(_ context.Context, userID string, limit, offset int32) ([]service_dto.Order, int64, error) {
	f.listCalled = true
	f.listCalledUserID = userID
	f.listCalledLimit = limit
	f.listCalledOffset = offset
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listOrders, f.listTotal, nil
}

type fakeOrderGateway struct {
	err error

	called     bool
	calledWith service_dto.Order

	ratedCalled     bool
	ratedCalledWith service_dto.Order
	ratedErr        error
}

func (f *fakeOrderGateway) PublishOrderCreated(_ context.Context, order service_dto.Order) error {
	f.called = true
	f.calledWith = order
	return f.err
}

func (f *fakeOrderGateway) PublishOrderRated(_ context.Context, order service_dto.Order) error {
	f.ratedCalled = true
	f.ratedCalledWith = order
	return f.ratedErr
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

type fakeDriverGateway struct {
	claimUserID string
	claimOK     bool
	claimErr    error
	releaseErr  error

	claimCalled   bool
	claimArg      string
	releaseCalled bool
	releaseArg    string
}

func (f *fakeDriverGateway) ClaimAvailableDriver(_ context.Context, taxiType string) (string, bool, error) {
	f.claimCalled = true
	f.claimArg = taxiType
	return f.claimUserID, f.claimOK, f.claimErr
}

func (f *fakeDriverGateway) ReleaseDriver(_ context.Context, userID string) error {
	f.releaseCalled = true
	f.releaseArg = userID
	return f.releaseErr
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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

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
	svc := NewOrderService(repo, gw, wallet, &fakeDriverGateway{})

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected CreateOrder to succeed despite publish failure, got %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}

func TestCreateOrder_AssignsDriverWhenAvailable(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", TaxiType: service_dto.TaxiTypeComfort, Status: service_dto.StatusCreated}
	assigned := created
	assigned.Status = service_dto.StatusDriverAssigned
	driverID := "driver-1"
	assigned.DriverID = &driverID

	repo := &fakeOrderRepository{order: &created, assignedOrder: &assigned}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{claimUserID: "driver-1", claimOK: true}
	svc := NewOrderService(repo, gw, wallet, driver)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != service_dto.StatusDriverAssigned {
		t.Fatalf("got status %q, want driver_assigned", got.Status)
	}
	if driver.claimArg != string(service_dto.TaxiTypeComfort) {
		t.Fatalf("got claim taxi type %q, want comfort", driver.claimArg)
	}
	if !repo.assignDriverCalled || repo.assignDriverArg != "driver-1" {
		t.Fatalf("expected repo.AssignDriver to be called with driver-1, got called=%v arg=%q", repo.assignDriverCalled, repo.assignDriverArg)
	}
	if gw.calledWith.Status != service_dto.StatusDriverAssigned {
		t.Fatalf("expected the published event to carry the assigned status, got %+v", gw.calledWith)
	}
}

func TestCreateOrder_NoDriverAvailable(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{claimOK: false}
	svc := NewOrderService(repo, gw, wallet, driver)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected no error when no driver is available, got %v", err)
	}
	if got.Status != service_dto.StatusCreated {
		t.Fatalf("got status %q, want created (unassigned)", got.Status)
	}
	if repo.assignDriverCalled {
		t.Fatal("AssignDriver must not be called when no driver was claimed")
	}
}

func TestCreateOrder_ClaimErrorDoesNotFailCreateOrder(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{claimErr: errors.New("driver service unreachable")}
	svc := NewOrderService(repo, gw, wallet, driver)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected CreateOrder to succeed despite a claim error, got %v", err)
	}
	if got.Status != service_dto.StatusCreated {
		t.Fatalf("got status %q, want created (unassigned)", got.Status)
	}
}

func TestCreateOrder_ReleasesDriverWhenAssignmentFails(t *testing.T) {
	created := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	repo := &fakeOrderRepository{order: &created, assignDriverErr: errors.New("db down")}
	gw := &fakeOrderGateway{}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{claimUserID: "driver-1", claimOK: true}
	svc := NewOrderService(repo, gw, wallet, driver)

	got, err := svc.CreateOrder(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected CreateOrder to succeed despite a failed assignment, got %v", err)
	}
	if got.Status != service_dto.StatusCreated {
		t.Fatalf("got status %q, want created (unassigned)", got.Status)
	}
	if !driver.releaseCalled || driver.releaseArg != "driver-1" {
		t.Fatalf("expected the claimed driver to be released, got called=%v arg=%q", driver.releaseCalled, driver.releaseArg)
	}
}
