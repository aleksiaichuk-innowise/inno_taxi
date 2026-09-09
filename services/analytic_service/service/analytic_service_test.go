package service

import (
	"context"
	"errors"
	"testing"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/errorsx"
)

func TestIngestOrderEvent(t *testing.T) {
	repo := &fakeOrderEventRepository{}
	svc := NewAnalyticService(repo)

	evt := service_dto.OrderEvent{OrderID: "order-1", UserID: "user-1", TaxiType: "economy", Status: "created", CreatedAt: time.Now()}
	if err := svc.IngestOrderEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.inserted) != 1 || repo.inserted[0].OrderID != "order-1" {
		t.Fatalf("expected the event to be inserted, got %+v", repo.inserted)
	}
}

func TestIngestUserRegisteredEvent(t *testing.T) {
	repo := &fakeOrderEventRepository{}
	svc := NewAnalyticService(repo)

	evt := service_dto.UserRegisteredEvent{UserID: "user-1", Name: "Jane Doe", Role: "driver", RegisteredAt: time.Now()}
	if err := svc.IngestUserRegisteredEvent(context.Background(), evt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.insertedUsers) != 1 || repo.insertedUsers[0].UserID != "user-1" {
		t.Fatalf("expected the event to be inserted, got %+v", repo.insertedUsers)
	}
}

func TestGetOrderStats_InvalidRange(t *testing.T) {
	repo := &fakeOrderEventRepository{}
	svc := NewAnalyticService(repo)

	now := time.Now()
	_, err := svc.GetOrderStats(context.Background(), now, now.AddDate(0, 0, -1))
	if !errors.Is(err, errorsx.ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestGetOrderStats_DelegatesToRepo(t *testing.T) {
	want := service_dto.OrderStats{TotalOrders: 5, CountsByStatus: map[string]int64{"created": 5}}
	repo := &fakeOrderEventRepository{stats: want}
	svc := NewAnalyticService(repo)

	now := time.Now()
	got, err := svc.GetOrderStats(context.Background(), now.AddDate(0, 0, -1), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalOrders != want.TotalOrders {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGetDailyOrderCounts_InvalidRange(t *testing.T) {
	repo := &fakeOrderEventRepository{}
	svc := NewAnalyticService(repo)

	now := time.Now()
	_, err := svc.GetDailyOrderCounts(context.Background(), now, now.AddDate(0, 0, -1))
	if !errors.Is(err, errorsx.ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestGetDailyOrderCounts_DelegatesToRepo(t *testing.T) {
	want := []service_dto.DailyOrderCount{{Date: "2026-09-01", Count: 3}}
	repo := &fakeOrderEventRepository{daily: want}
	svc := NewAnalyticService(repo)

	now := time.Now()
	got, err := svc.GetDailyOrderCounts(context.Background(), now.AddDate(0, 0, -1), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Date != "2026-09-01" {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
