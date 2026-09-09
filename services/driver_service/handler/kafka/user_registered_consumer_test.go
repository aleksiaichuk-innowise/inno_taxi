package kafka

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	input "github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/driver_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"google.golang.org/protobuf/proto"
)

type fakeDriverRepository struct {
	createCalled bool
	createArg    *input.CreateDriverInput
	createErr    error
}

func (f *fakeDriverRepository) CreateDriver(_ context.Context, dto *input.CreateDriverInput) (input.Driver, error) {
	f.createCalled = true
	f.createArg = dto
	if f.createErr != nil {
		return input.Driver{}, f.createErr
	}
	return input.Driver{UserID: dto.UserID, TaxiType: dto.TaxiType, Status: input.StatusOffline}, nil
}

func (f *fakeDriverRepository) FindByUserID(context.Context, string) (input.Driver, error) {
	return input.Driver{}, errorsx.ErrDriverNotFound
}
func (f *fakeDriverRepository) UpdateStatusByUserID(context.Context, string, string) error {
	return nil
}
func (f *fakeDriverRepository) UpdateTaxiTypeByUserID(context.Context, string, string) error {
	return nil
}
func (f *fakeDriverRepository) FindByStatus(context.Context, input.Status) ([]input.Driver, error) {
	return nil, nil
}

func mustMarshal(t *testing.T, event *userpb.UserRegisteredEvent) []byte {
	t.Helper()
	b, err := proto.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return b
}

func TestUserRegisteredConsumer_CreatesDriverForDriverRole(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := service.NewDriverService(repo)
	c := NewUserRegisteredConsumer(svc)

	value := mustMarshal(t, &userpb.UserRegisteredEvent{UserId: "user-1", Role: "driver"})
	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if !repo.createCalled {
		t.Fatal("expected CreateDriver to be called")
	}
	if repo.createArg.UserID != "user-1" {
		t.Errorf("got user ID %q, want %q", repo.createArg.UserID, "user-1")
	}
	if repo.createArg.TaxiType != defaultTaxiType {
		t.Errorf("got taxi type %q, want default %q", repo.createArg.TaxiType, defaultTaxiType)
	}
}

func TestUserRegisteredConsumer_SkipsNonDriverRole(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := service.NewDriverService(repo)
	c := NewUserRegisteredConsumer(svc)

	value := mustMarshal(t, &userpb.UserRegisteredEvent{UserId: "user-1", Role: "user"})
	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if repo.createCalled {
		t.Fatal("expected CreateDriver not to be called for a non-driver role")
	}
}

func TestUserRegisteredConsumer_IgnoresAlreadyExists(t *testing.T) {
	repo := &fakeDriverRepository{createErr: errorsx.ErrDriverAlreadyExists}
	svc := service.NewDriverService(repo)
	c := NewUserRegisteredConsumer(svc)

	value := mustMarshal(t, &userpb.UserRegisteredEvent{UserId: "user-1", Role: "driver"})
	// Must not panic; ErrDriverAlreadyExists is a benign, expected outcome
	// under at-least-once redelivery.
	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if !repo.createCalled {
		t.Fatal("expected CreateDriver to be called")
	}
}

func TestUserRegisteredConsumer_SkipsUnparsableMessage(t *testing.T) {
	repo := &fakeDriverRepository{}
	svc := service.NewDriverService(repo)
	c := NewUserRegisteredConsumer(svc)

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: []byte("not-a-proto-message")})

	if repo.createCalled {
		t.Fatal("expected CreateDriver not to be called for an unparsable message")
	}
}
