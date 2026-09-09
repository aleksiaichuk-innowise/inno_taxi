package kafka

import (
	"context"
	"errors"
	"log/slog"

	"github.com/IBM/sarama"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

const TopicUserRegistered = "user_registered"

// UserRegisteredConsumer implements sarama.ConsumerGroupHandler. A message
// that fails to unmarshal or ingest is logged and marked processed anyway
// (never retried) - same trade-off as analytic_service's order_created
// consumer: there's no dead-letter queue here, and blocking the partition
// on a permanently malformed message would stall everything behind it.
type UserRegisteredConsumer struct {
	svc *service.WalletService
}

func NewUserRegisteredConsumer(svc *service.WalletService) *UserRegisteredConsumer {
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

	// Every registration gets a wallet, regardless of role (user or driver).
	_, err := c.svc.CreateWallet(ctx, event.GetUserId())
	if err != nil {
		if errors.Is(err, errorsx.ErrWalletAlreadyExists) {
			slog.InfoContext(ctx, "wallet already exists, skipping", "user_id", event.GetUserId())
			return
		}
		slog.ErrorContext(ctx, "create wallet from user registered event", "error", err, "user_id", event.GetUserId())
	}
}
