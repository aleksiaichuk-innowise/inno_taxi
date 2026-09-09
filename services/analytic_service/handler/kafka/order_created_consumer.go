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

const TopicOrderCreated = "order_created"

// OrderCreatedConsumer implements sarama.ConsumerGroupHandler. A message
// that fails to unmarshal or ingest is logged and marked processed anyway
// (never retried) - Kafka has no dead-letter queue wired up here, and a
// message that's malformed once will be malformed on every retry too;
// blocking the whole partition on it would just stall every event behind
// it, permanently, for no gain.
type OrderCreatedConsumer struct {
	svc *service.AnalyticService
}

func NewOrderCreatedConsumer(svc *service.AnalyticService) *OrderCreatedConsumer {
	return &OrderCreatedConsumer{svc: svc}
}

func (c *OrderCreatedConsumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *OrderCreatedConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *OrderCreatedConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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

func (c *OrderCreatedConsumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) {
	var event orderpb.OrderCreatedEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "unmarshal order created event", "error", err, "offset", msg.Offset)
		return
	}

	createdAt, err := time.Parse(time.RFC3339, event.GetCreatedAt())
	if err != nil {
		slog.ErrorContext(ctx, "parse order created_at", "error", err, "order_id", event.GetOrderId())
		return
	}

	evt := service_dto.OrderEvent{
		OrderID:   event.GetOrderId(),
		UserID:    event.GetUserId(),
		TaxiType:  taxiTypeString(event.GetTaxiType()),
		Status:    statusString(event.GetStatus()),
		CreatedAt: createdAt,
	}

	if err := c.svc.IngestOrderEvent(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "ingest order event", "error", err, "order_id", evt.OrderID)
	}
}
