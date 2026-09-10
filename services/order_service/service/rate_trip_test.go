package service

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/errorsx"
)

func TestRateTrip_InvalidRating(t *testing.T) {
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	for _, r := range []int32{0, 6, -1} {
		_, err := svc.RateTrip(context.Background(), "order-1", "user-1", r, nil)
		if !errors.Is(err, errorsx.ErrInvalidRating) {
			t.Fatalf("rating=%d: expected ErrInvalidRating, got %v", r, err)
		}
	}
}

func TestRateTrip_NotFound(t *testing.T) {
	repo := &fakeOrderRepository{getErr: errorsx.ErrOrderNotFound}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestRateTrip_NotYourOrder(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "someone-else", Status: service_dto.StatusCompleted}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if !errors.Is(err, errorsx.ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound (not distinguishing from not-yours), got %v", err)
	}
}

func TestRateTrip_NotCompleted(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusInProgress}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if !errors.Is(err, errorsx.ErrOrderNotRatable) {
		t.Fatalf("expected ErrOrderNotRatable, got %v", err)
	}
}

func TestRateTrip_AlreadyRated(t *testing.T) {
	existing := int32(4)
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted, Rating: &existing}
	repo := &fakeOrderRepository{getOrder: &order}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if !errors.Is(err, errorsx.ErrOrderAlreadyRated) {
		t.Fatalf("expected ErrOrderAlreadyRated, got %v", err)
	}
	if repo.rateOrderCalled {
		t.Fatal("RateOrder must not be called when already rated (fast path)")
	}
}

func TestRateTrip_RaceCaughtByRepository(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	repo := &fakeOrderRepository{getOrder: &order, rateOrderErr: errorsx.ErrOrderAlreadyRated}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if !errors.Is(err, errorsx.ErrOrderAlreadyRated) {
		t.Fatalf("expected ErrOrderAlreadyRated surfaced from the repository, got %v", err)
	}
}

func TestRateTrip_Succeeds(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	rating := int32(5)
	comment := "great ride"
	rated := order
	rated.Rating = &rating
	rated.Comment = &comment
	repo := &fakeOrderRepository{getOrder: &order, ratedOrder: &rated}
	gw := &fakeOrderGateway{}
	svc := NewOrderService(repo, gw, &fakeWalletGateway{}, &fakeDriverGateway{})

	got, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, &comment)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Rating == nil || *got.Rating != 5 {
		t.Fatalf("got rating %v, want 5", got.Rating)
	}
	if !gw.ratedCalled {
		t.Fatal("expected gateway.PublishOrderRated to be called")
	}
}

func TestRateTrip_PublishFailureDoesNotFailRateTrip(t *testing.T) {
	order := service_dto.Order{ID: "order-1", UserID: "user-1", Status: service_dto.StatusCompleted}
	repo := &fakeOrderRepository{getOrder: &order}
	gw := &fakeOrderGateway{ratedErr: errors.New("kafka unreachable")}
	svc := NewOrderService(repo, gw, &fakeWalletGateway{}, &fakeDriverGateway{})

	_, err := svc.RateTrip(context.Background(), "order-1", "user-1", 5, nil)
	if err != nil {
		t.Fatalf("expected RateTrip to succeed despite publish failure, got %v", err)
	}
}
