package service

import (
	"context"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func TestListOrders_DefaultsLimit(t *testing.T) {
	repo := &fakeOrderRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	if _, _, err := svc.ListOrders(context.Background(), "user-1", 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listCalledLimit != defaultListOrdersLimit {
		t.Fatalf("got limit %d, want default %d", repo.listCalledLimit, defaultListOrdersLimit)
	}
	if repo.listCalledOffset != 0 {
		t.Fatalf("got offset %d, want 0", repo.listCalledOffset)
	}
}

func TestListOrders_ClampsOversizedLimit(t *testing.T) {
	repo := &fakeOrderRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	if _, _, err := svc.ListOrders(context.Background(), "user-1", 10000, -5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listCalledLimit != maxListOrdersLimit {
		t.Fatalf("got limit %d, want cap %d", repo.listCalledLimit, maxListOrdersLimit)
	}
	if repo.listCalledOffset != 0 {
		t.Fatalf("got offset %d, want clamped to 0", repo.listCalledOffset)
	}
}

func TestListOrders_PassesThroughValidValues(t *testing.T) {
	repo := &fakeOrderRepository{}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	if _, _, err := svc.ListOrders(context.Background(), "user-1", 10, 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listCalledLimit != 10 || repo.listCalledOffset != 30 {
		t.Fatalf("got limit=%d offset=%d, want 10/30", repo.listCalledLimit, repo.listCalledOffset)
	}
	if repo.listCalledUserID != "user-1" {
		t.Fatalf("got user id %q, want user-1", repo.listCalledUserID)
	}
}

func TestListOrders_DelegatesResults(t *testing.T) {
	want := []service_dto.Order{{ID: "order-1"}, {ID: "order-2"}}
	repo := &fakeOrderRepository{listOrders: want, listTotal: 5}
	svc := NewOrderService(repo, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{})

	got, total, err := svc.ListOrders(context.Background(), "user-1", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || total != 5 {
		t.Fatalf("got %d orders, total=%d, want 2/5", len(got), total)
	}
}
