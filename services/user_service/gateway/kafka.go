package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

const topicUserRegistered = "user_registered"

type KafkaGateway interface {
	PublishUserRegistered(ctx context.Context, user service_dto.User) error
}

type kafkaGateway struct {
	producer sarama.SyncProducer
}

func NewKafkaGateway(producer sarama.SyncProducer) KafkaGateway {
	return &kafkaGateway{producer: producer}
}

func (k *kafkaGateway) PublishUserRegistered(ctx context.Context, user service_dto.User) error {
	var role string
	if len(user.Roles) > 0 {
		role = string(user.Roles[0])
	}

	event := &userpb.UserRegisteredEvent{
		UserId:    user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Role:      role,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}

	value, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal user registered event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topicUserRegistered,
		Key:   sarama.StringEncoder(user.ID),
		Value: sarama.ByteEncoder(value),
	}

	_, _, err = k.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("publish user registered event: %w", err)
	}
	return nil
}
