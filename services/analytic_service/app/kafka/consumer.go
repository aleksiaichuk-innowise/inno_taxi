package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
)

const ConsumerGroupID = "analytic_service"

func NewConsumerGroup(brokers []string) (sarama.ConsumerGroup, error) {
	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Return.Errors = true

	group, err := sarama.NewConsumerGroup(brokers, ConsumerGroupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("new consumer group: %w", err)
	}
	return group, nil
}
