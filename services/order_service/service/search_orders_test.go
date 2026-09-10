package service

import (
	"context"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
)

func TestSearchOrders_DefaultsLimitWhenUnset(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Limit != defaultListOrdersLimit {
		t.Fatalf("got limit %d, want default %d", search.searchCalledWith.Limit, defaultListOrdersLimit)
	}
}

func TestSearchOrders_CapsLimitAtMax(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Limit: 1000}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Limit != maxListOrdersLimit {
		t.Fatalf("got limit %d, want cap %d", search.searchCalledWith.Limit, maxListOrdersLimit)
	}
}

func TestSearchOrders_NegativeOffsetClampedToZero(t *testing.T) {
	search := &fakeSearchRepository{}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	if _, _, err := svc.SearchOrders(context.Background(), service_dto.OrderSearchFilter{Offset: -5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if search.searchCalledWith.Offset != 0 {
		t.Fatalf("got offset %d, want 0", search.searchCalledWith.Offset)
	}
}

func TestSearchOrders_PassesThroughFilterAndResults(t *testing.T) {
	want := []service_dto.Order{{ID: "order-1"}}
	search := &fakeSearchRepository{searchOrders: want, searchTotal: 1}
	svc := NewOrderService(&fakeOrderRepository{}, &fakeOrderGateway{}, &fakeWalletGateway{}, &fakeDriverGateway{}, search)

	filter := service_dto.OrderSearchFilter{TaxiType: service_dto.TaxiTypeComfort, Limit: 10}
	orders, total, err := svc.SearchOrders(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(orders) != 1 || orders[0].ID != "order-1" {
		t.Fatalf("unexpected results: orders=%+v total=%d", orders, total)
	}
	if search.searchCalledWith.TaxiType != service_dto.TaxiTypeComfort {
		t.Fatalf("expected the filter to be passed through, got %+v", search.searchCalledWith)
	}
}
