package gateway

import (
	"context"
	"errors"
	"testing"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/order_service/entity/service"
	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"google.golang.org/protobuf/proto"

	"github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/order_service"
)

func TestKafkaGateway_PublishOrderCreated_Succeeds(t *testing.T) {
	order := service_dto.Order{
		ID:       "order-1",
		UserID:   "user-1",
		TaxiType: service_dto.TaxiTypeEconomy,
		Status:   service_dto.StatusCreated,
	}

	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
		if msg.Topic != topicOrderCreated {
			return errors.New("unexpected topic: " + msg.Topic)
		}
		key, err := msg.Key.Encode()
		if err != nil {
			return err
		}
		if string(key) != order.ID {
			return errors.New("unexpected key: " + string(key))
		}
		value, err := msg.Value.Encode()
		if err != nil {
			return err
		}
		var event order_service.OrderCreatedEvent
		if err := proto.Unmarshal(value, &event); err != nil {
			return err
		}
		if event.GetOrderId() != order.ID {
			return errors.New("unexpected order id in payload: " + event.GetOrderId())
		}
		return nil
	})
	defer func() {
		if err := producer.Close(); err != nil {
			t.Fatalf("close producer: %v", err)
		}
	}()

	gw := NewKafkaGateway(producer)
	if err := gw.PublishOrderCreated(context.Background(), order); err != nil {
		t.Fatalf("PublishOrderCreated() error = %v, want nil", err)
	}
}

func TestKafkaGateway_PublishOrderCreated_ReturnsProducerError(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	wantErr := errors.New("broker unreachable")
	producer.ExpectSendMessageAndFail(wantErr)
	defer func() {
		if err := producer.Close(); err != nil {
			t.Fatalf("close producer: %v", err)
		}
	}()

	gw := NewKafkaGateway(producer)
	err := gw.PublishOrderCreated(context.Background(), service_dto.Order{ID: "order-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("PublishOrderCreated() error = %v, want wrapping %v", err, wantErr)
	}
}
