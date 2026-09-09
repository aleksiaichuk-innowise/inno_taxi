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
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet)

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
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet)

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
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet)

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
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet)

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
	svc := NewOrderService(repo, &fakeOrderGateway{}, wallet)

	_, err := svc.CancelOrder(context.Background(), "order-1", "user-1")
	if err == nil {
		t.Fatal("expected an error when the refund fails")
	}
	if repo.updateCalled {
		t.Fatal("status must not be updated when the refund fails")
	}
}
