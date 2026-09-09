package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/analytic_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

const TopicUserRegistered = "user_registered"

// UserRegisteredConsumer implements sarama.ConsumerGroupHandler, same
// mark-always/log-and-skip shape as OrderCreatedConsumer - see that file's
// comment for the reasoning.
type UserRegisteredConsumer struct {
	svc *service.AnalyticService
}

func NewUserRegisteredConsumer(svc *service.AnalyticService) *UserRegisteredConsumer {
	return &UserRegisteredConsumer{svc: svc}
}

func (c *UserRegisteredConsumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *UserRegisteredConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *UserRegisteredConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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

func (c *UserRegisteredConsumer) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) {
	var event userpb.UserRegisteredEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "unmarshal user registered event", "error", err, "offset", msg.Offset)
		return
	}

	registeredAt, err := time.Parse(time.RFC3339, event.GetCreatedAt())
	if err != nil {
		slog.ErrorContext(ctx, "parse user registered created_at", "error", err, "user_id", event.GetUserId())
		return
	}

	evt := service_dto.UserRegisteredEvent{
		UserID:       event.GetUserId(),
		Name:         event.GetName(),
		Email:        event.GetEmail(),
		Phone:        event.GetPhone(),
		Role:         event.GetRole(),
		RegisteredAt: registeredAt,
	}

	if err := c.svc.IngestUserRegisteredEvent(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "ingest user registered event", "error", err, "user_id", evt.UserID)
	}
}
