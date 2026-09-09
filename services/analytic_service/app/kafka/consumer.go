package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
)

// Two independent consumer groups, one per topic, rather than one group
// subscribed to both topics - keeps order_created's already-working
// consumption (and its committed offsets/group identity) untouched by the
// new user_registered wiring.
const (
	ConsumerGroupIDOrderCreated   = "analytic_service"
	ConsumerGroupIDUserRegistered = "analytic_service_user_registered"
)

func NewConsumerGroup(brokers []string, groupID string) (sarama.ConsumerGroup, error) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Return.Errors = true

	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("new consumer group: %w", err)
	}
	return group, nil
}
