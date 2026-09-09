package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/user_service/entity/service"
	"google.golang.org/protobuf/proto"

	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
)

func TestKafkaGateway_PublishUserRegistered_Succeeds(t *testing.T) {
	user := service_dto.User{
		ID:        "user-1",
		Name:      "Jane Doe",
		Email:     "jane@example.com",
		Phone:     "+15550001111",
		Roles:     []service_dto.Role{service_dto.RoleDriver},
		CreatedAt: time.Now(),
	}

	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
		if msg.Topic != topicUserRegistered {
			return errors.New("unexpected topic: " + msg.Topic)
		}
		key, err := msg.Key.Encode()
		if err != nil {
			return err
		}
		if string(key) != user.ID {
			return errors.New("unexpected key: " + string(key))
		}
		value, err := msg.Value.Encode()
		if err != nil {
			return err
		}
		var event userpb.UserRegisteredEvent
		if err := proto.Unmarshal(value, &event); err != nil {
			return err
		}
		if event.GetUserId() != user.ID {
			return errors.New("unexpected user id in payload: " + event.GetUserId())
		}
		if event.GetRole() != "driver" {
			return errors.New("unexpected role in payload: " + event.GetRole())
		}
		return nil
	})
	defer func() {
		if err := producer.Close(); err != nil {
			t.Fatalf("close producer: %v", err)
		}
	}()

	gw := NewKafkaGateway(producer)
	if err := gw.PublishUserRegistered(context.Background(), user); err != nil {
		t.Fatalf("PublishUserRegistered() error = %v, want nil", err)
	}
}

func TestKafkaGateway_PublishUserRegistered_ReturnsProducerError(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	wantErr := errors.New("broker unreachable")
	producer.ExpectSendMessageAndFail(wantErr)
	defer func() {
		if err := producer.Close(); err != nil {
			t.Fatalf("close producer: %v", err)
		}
	}()

	gw := NewKafkaGateway(producer)
	err := gw.PublishUserRegistered(context.Background(), service_dto.User{ID: "user-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("PublishUserRegistered() error = %v, want wrapping %v", err, wantErr)
	}
}
