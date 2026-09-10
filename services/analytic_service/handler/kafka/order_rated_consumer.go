package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"
	orderpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
	"google.golang.org/protobuf/proto"
)

const TopicOrderRated = "order_rated"

// OrderRatedConsumer implements sarama.ConsumerGroupHandler, same
// mark-always/log-and-skip shape as OrderCreatedConsumer.
type OrderRatedConsumer struct {
	svc *service.AnalyticService
}

func NewOrderRatedConsumer(svc *service.AnalyticService) *OrderRatedConsumer {
	return &OrderRatedConsumer{svc: svc}
}

func (c *OrderRatedConsumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *OrderRatedConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *OrderRatedConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			c.handleMessage(session.Context(), msg)
			session.MarkMessage(msg, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

func (c *OrderRatedConsumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) {
	var event orderpb.OrderRatedEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "unmarshal order rated event", "error", err, "offset", msg.Offset)
		return
	}

	ratedAt, err := time.Parse(time.RFC3339, event.GetRatedAt())
	if err != nil {
		slog.ErrorContext(ctx, "parse order rated_at", "error", err, "order_id", event.GetOrderId())
		return
	}

	evt := service_dto.DriverRatingEvent{
		OrderID:  event.GetOrderId(),
		DriverID: event.GetDriverId(),
		Rating:   event.GetRating(),
		Comment:  event.GetComment(),
		RatedAt:  ratedAt,
	}

	if err := c.svc.IngestDriverRating(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "ingest driver rating", "error", err, "order_id", evt.OrderID)
	}
}
