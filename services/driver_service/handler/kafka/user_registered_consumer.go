package kafka

import (
	"context"
	"errors"
	"log/slog"

	"github.com/IBM/sarama"
	input "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

const TopicUserRegistered = "user_registered"

// defaultTaxiType is used for auto-created driver profiles: registration
// (POST /register on user_service) doesn't collect a taxi type, so this is
// the starting value until the driver changes it via PATCH /profile.
const defaultTaxiType = input.TaxiTypeEconomy

// UserRegisteredConsumer implements sarama.ConsumerGroupHandler. A message
// that fails to unmarshal or ingest is logged and marked processed anyway
// (never retried) - same trade-off as analytic_service's order_created
// consumer: there's no dead-letter queue here, and blocking the partition
// on a permanently malformed message would stall everything behind it.
type UserRegisteredConsumer struct {
	svc *service.DriverService
}

func NewUserRegisteredConsumer(svc *service.DriverService) *UserRegisteredConsumer {
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

	if event.GetRole() != "driver" {
		return
	}

	_, err := c.svc.CreateDriver(ctx, &input.CreateDriverInput{
		UserID:   event.GetUserId(),
		TaxiType: defaultTaxiType,
	})
	if err != nil {
		if errors.Is(err, errorsx.ErrDriverAlreadyExists) {
			slog.InfoContext(ctx, "driver profile already exists, skipping", "user_id", event.GetUserId())
			return
		}
		slog.ErrorContext(ctx, "create driver from user registered event", "error", err, "user_id", event.GetUserId())
	}
}
