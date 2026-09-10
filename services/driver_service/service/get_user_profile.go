package service

import (
	"context"
	"log/slog"

	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
)

// GetProfileByUser enriches the stored driver profile with rating stats read
// live from analytic_service. That read is best-effort: analytic_service
// being unreachable is a display-data gap, not a reason to fail a profile
// fetch that otherwise succeeded.
func (s DriverService) GetProfileByUser(ctx context.Context, id string) (service.DriverProfile, error) {
	d, err := s.repo.FindByUserID(ctx, id)
	if err != nil {
		return service.DriverProfile{}, err
	}

	profile := service.DriverProfile{Driver: d}

	average, count, err := s.ratings.GetDriverRatingStats(ctx, d.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "get driver rating stats failed", "user_id", d.UserID, "error", err)
		return profile, nil
	}
	profile.RatingAverage = average
	profile.RatingCount = count

	return profile, nil
}
