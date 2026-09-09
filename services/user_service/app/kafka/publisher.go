package kafka

import (
	"time"

	"github.com/IBM/sarama"
)

func NewPublisher(brokers []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Idempotent = true
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Retry.Backoff = 500 * time.Millisecond
	cfg.Producer.Return.Successes = true
	cfg.Net.MaxOpenRequests = 1

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return p, nil
}
