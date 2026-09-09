package gateway

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"google.golang.org/protobuf/proto"
)

const topicOrderCreated = "order_created"

type KafkaGateway interface {
	PublishOrderCreated(ctx context.Context, order service_dto.Order) error
}

type kafkaGateway struct {
	producer sarama.SyncProducer
}

func NewKafkaGateway(producer sarama.SyncProducer) KafkaGateway {
	return &kafkaGateway{producer: producer}
}

func (k *kafkaGateway) PublishOrderCreated(ctx context.Context, order service_dto.Order) error {
	event := orderCreatedEventFromOrder(order)
	value, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order created event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topicOrderCreated,
		Key:   sarama.StringEncoder(order.ID),
		Value: sarama.ByteEncoder(value),
	}

	_, _, err = k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("publish order created event: %w", err)
	}
	return nil
}
