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

	insertedUsers []service_dto.UserRegisteredEvent

	insertedRatings []service_dto.DriverRatingEvent
	ratingStats     service_dto.DriverRatingStats
	ratingStatsErr  error
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

func (f *fakeOrderEventRepository) InsertUserRegisteredEvent(_ context.Context, evt service_dto.UserRegisteredEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertedUsers = append(f.insertedUsers, evt)
	return nil
}

func (f *fakeOrderEventRepository) InsertDriverRating(_ context.Context, evt service_dto.DriverRatingEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertedRatings = append(f.insertedRatings, evt)
	return nil
}

func (f *fakeOrderEventRepository) GetDriverRatingStats(_ context.Context, _ string) (service_dto.DriverRatingStats, error) {
	if f.ratingStatsErr != nil {
		return service_dto.DriverRatingStats{}, f.ratingStatsErr
	}
	return f.ratingStats, nil
}
