package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

type fakeEventRepository struct {
	insertedUsers   []service_dto.UserRegisteredEvent
	insertedRatings []service_dto.DriverRatingEvent
}

func (f *fakeEventRepository) InsertOrderEvent(context.Context, service_dto.OrderEvent) error {
	return nil
}
func (f *fakeEventRepository) GetOrderStats(context.Context, time.Time, time.Time) (service_dto.OrderStats, error) {
	return service_dto.OrderStats{}, nil
}
func (f *fakeEventRepository) GetDailyOrderCounts(context.Context, time.Time, time.Time) ([]service_dto.DailyOrderCount, error) {
	return nil, nil
}
func (f *fakeEventRepository) InsertUserRegisteredEvent(_ context.Context, evt service_dto.UserRegisteredEvent) error {
	f.insertedUsers = append(f.insertedUsers, evt)
	return nil
}
func (f *fakeEventRepository) InsertDriverRating(_ context.Context, evt service_dto.DriverRatingEvent) error {
	f.insertedRatings = append(f.insertedRatings, evt)
	return nil
}
func (f *fakeEventRepository) GetDriverRatingStats(context.Context, string) (service_dto.DriverRatingStats, error) {
	return service_dto.DriverRatingStats{}, nil
}

func TestUserRegisteredConsumer_IngestsValidEvent(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewUserRegisteredConsumer(svc)

	createdAt := time.Now().UTC().Truncate(time.Second)
	value, err := proto.Marshal(&userpb.UserRegisteredEvent{
		UserId:    "user-1",
		Name:      "Jane Doe",
		Role:      "driver",
		CreatedAt: createdAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if len(repo.insertedUsers) != 1 {
		t.Fatalf("got %d inserted events, want 1", len(repo.insertedUsers))
	}
	got := repo.insertedUsers[0]
	if got.UserID != "user-1" || got.Role != "driver" {
		t.Errorf("unexpected inserted event: %+v", got)
	}
	if !got.RegisteredAt.Equal(createdAt) {
		t.Errorf("got registered_at %v, want %v", got.RegisteredAt, createdAt)
	}
}

func TestUserRegisteredConsumer_SkipsUnparsableMessage(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewUserRegisteredConsumer(svc)

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: []byte("not-a-proto-message")})

	if len(repo.insertedUsers) != 0 {
		t.Fatal("expected no event to be inserted for an unparsable message")
	}
}

func TestUserRegisteredConsumer_SkipsInvalidCreatedAt(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewUserRegisteredConsumer(svc)

	value, err := proto.Marshal(&userpb.UserRegisteredEvent{UserId: "user-1", Role: "driver", CreatedAt: "not-a-timestamp"})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if len(repo.insertedUsers) != 0 {
		t.Fatal("expected no event to be inserted for an unparsable created_at")
	}
}
