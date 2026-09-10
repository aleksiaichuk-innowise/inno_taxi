package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func priceP(v int64) *int64 { return &v }

func TestCancelOrder_NotFound(t *testing.T) {
	repo := &fakeOrderRepository{getErr: errorsx.ErrOrderNotFound}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, &fakeDriverGateway{}, nil)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
	if wallet.refundCalled {
		t.Fatal("wallet must not be refunded when the order doesn't exist")
	}
}

func TestCancelOrder_NotOwner(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "someone-else", Status: service_dto.StatusCreated, PriceMinorUnits: priceP(500)}
	repo := &fakeOrderRepository{getOrder: &order}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, &fakeDriverGateway{}, nil)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound (not distinguishing from not-yours), got %v", err)
	}
	if wallet.refundCalled {
		t.Fatal("wallet must not be refunded for someone else's order")
	}
}

func TestCancelOrder_AlreadyCancelled(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCancelled, PriceMinorUnits: priceP(500)}
	repo := &fakeOrderRepository{getOrder: &order}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, &fakeDriverGateway{}, nil)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if !errors.Is(err, errorsx.ErrOrderNotCancellable) {
		t.Fatalf("expected ErrOrderNotCancellable, got %v", err)
	}
	if wallet.refundCalled {
		t.Fatal("wallet must not be refunded for an order that's already cancelled")
	}
}

func TestCancelOrder_RefundsThenUpdatesStatus(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated, PriceMinorUnits: priceP(800)}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	wallet := &fakeWalletGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, &fakeDriverGateway{}, &fakeSearchRepository{})

	got, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != service_dto.StatusCancelled {
		t.Fatalf("got status %q, want cancelled", got.Status)
	}
	if !wallet.refundCalled {
		t.Fatal("expected wallet.Refund to be called")
	}
	if wallet.refundArgs[0] != "user-1" || wallet.refundArgs[1] != int64(800) || wallet.refundArgs[2] != "order-1" {
		t.Fatalf("unexpected refund args: %+v", wallet.refundArgs)
	}
}

func TestCancelOrder_RefundFailureBlocksStatusUpdate(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated, PriceMinorUnits: priceP(800)}
	repo := &fakeOrderRepository{getOrder: &order}
	wallet := &fakeWalletGateway{refundErr: errors.New("wallet unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, &fakeDriverGateway{}, nil)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err == nil {
		t.Fatal("expected an error when the refund fails")
	}
	if repo.updateCalled {
		t.Fatal("status must not be updated when the refund fails")
	}
}

func TestCancelOrder_AllowsDriverAssignedStatus(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID, PriceMinorUnits: priceP(800)}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, driver, &fakeSearchRepository{})

	got, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != service_dto.StatusCancelled {
		t.Fatalf("got status %q, want cancelled", got.Status)
	}
	if !driver.releaseCalled || driver.releaseArg != "driver-1" {
		t.Fatalf("expected the assigned driver to be released, got called=%v arg=%q", driver.releaseCalled, driver.releaseArg)
	}
}

func TestCancelOrder_NoDriverReleaseWhenUnassigned(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated, PriceMinorUnits: priceP(800)}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, driver, &fakeSearchRepository{})

	if _, err := svc.CancelOrder(context.Background(), "order-1", "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if driver.releaseCalled {
		t.Fatal("driver.ReleaseDriver must not be called for an order with no assigned driver")
	}
}

func TestCancelOrder_ReleaseFailureDoesNotBlockCancellation(t *testing.T) {
	driverID := "driver-1"
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusDriverAssigned, DriverID: &driverID, PriceMinorUnits: priceP(800)}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	wallet := &fakeWalletGateway{}
	driver := &fakeDriverGateway{releaseErr: errors.New("driver service unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet, driver, &fakeSearchRepository{})

	got, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("expected cancellation to succeed despite a release failure, got %v", err)
	}
	if got.Status != service_dto.StatusCancelled {
		t.Fatalf("got status %q, want cancelled", got.Status)
	}
}

func TestCancelOrder_IndexesOrderAfterCancelling(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	search := &fakeSearchRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !search.indexCalled || search.indexCalledWith.Status != service_dto.StatusCancelled {
		t.Fatalf("expected the cancelled order to be indexed, got called=%v order=%+v", search.indexCalled, search.indexCalledWith)
	}
}

func TestCancelOrder_IndexFailureDoesNotFailCancelOrder(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCreated}
	cancelled := order
	cancelled.Status = service_dto.StatusCancelled
	repo := &fakeOrderRepository{getOrder: &order, updateOrder: &cancelled}
	search := &fakeSearchRepository{indexErr: errors.New("elasticsearch unreachable")}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
