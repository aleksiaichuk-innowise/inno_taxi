package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"
	orderpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
	"google.golang.org/protobuf/proto"
)

func TestOrderRatedConsumer_IngestsValidEvent(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewOrderRatedConsumer(svc)

	ratedAt := time.Now().UTC().Truncate(time.Second)
	value, err := proto.Marshal(&orderpb.OrderRatedEvent{
		OrderId:  "order-1",
		DriverId: "driver-1",
		Rating:   5,
		Comment:  "great ride",
		RatedAt:  ratedAt.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if len(repo.insertedRatings) != 1 {
		t.Fatalf("got %d inserted ratings, want 1", len(repo.insertedRatings))
	}
	got := repo.insertedRatings[0]
	if got.OrderID != "order-1" || got.DriverID != "driver-1" || got.Rating != 5 {
		t.Errorf("unexpected inserted rating: %+v", got)
	}
	if !got.RatedAt.Equal(ratedAt) {
		t.Errorf("got rated_at %v, want %v", got.RatedAt, ratedAt)
	}
}

func TestOrderRatedConsumer_SkipsUnparsableMessage(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewOrderRatedConsumer(svc)

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: []byte("not-a-proto-message")})

	if len(repo.insertedRatings) != 0 {
		t.Fatal("expected no rating to be inserted for an unparsable message")
	}
}

func TestOrderRatedConsumer_SkipsInvalidRatedAt(t *testing.T) {
	repo := &fakeEventRepository{}
	svc := service.NewAnalyticService(repo)
	c := NewOrderRatedConsumer(svc)

	value, err := proto.Marshal(&orderpb.OrderRatedEvent{OrderId: "order-1", RatedAt: "not-a-timestamp"})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if len(repo.insertedRatings) != 0 {
		t.Fatal("expected no rating to be inserted for an unparsable rated_at")
	}
}
