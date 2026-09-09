package service

import (
	"context"
	"sync"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
)

type fakeOrderEventRepository struct {
	mu       sync.Mutex
	inserted []service_dto.OrderEvent

	stats    service_dto.OrderStats
	daily    []service_dto.DailyOrderCount
	statsErr error
	dailyErr error
}

func (f *fakeOrderEventRepository) InsertOrderEvent(_ context.Context, evt service_dto.OrderEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inserted = append(f.inserted, evt)
	return nil
}

func (f *fakeOrderEventRepository) GetOrderStats(_ context.Context, _, _ time.Time) (service_dto.OrderStats, error) {
	if f.statsErr != nil {
		return service_dto.OrderStats{}, f.statsErr
	}
	return f.stats, nil
}

func (f *fakeOrderEventRepository) GetDailyOrderCounts(_ context.Context, _, _ time.Time) ([]service_dto.DailyOrderCount, error) {
	if f.dailyErr != nil {
		return nil, f.dailyErr
	}
	return f.daily, nil
}
