package kafka

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/service"
	userpb "github.com/aleksiaichuk-innowise/inno_taxi/shared/proto/user_service"
	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/proto"
)

type fakeWalletRepository struct {
	createCalled bool
	createArg    string
	createErr    error
}

func (f *fakeWalletRepository) CreateWallet(_ context.Context, userID string) (service_dto.Wallet, error) {
	f.createCalled = true
	f.createArg = userID
	if f.createErr != nil {
		return service_dto.Wallet{}, f.createErr
	}
	return service_dto.Wallet{ID: "wallet-" + userID, UserID: userID}, nil
}

func (f *fakeWalletRepository) GetWalletByUserID(context.Context, string) (service_dto.Wallet, error) {
	return service_dto.Wallet{}, errorsx.ErrWalletNotFound
}
func (f *fakeWalletRepository) Start(context.Context) (*sqlx.Tx, error) { return nil, nil }
func (f *fakeWalletRepository) Finish(*sqlx.Tx) error                   { return nil }
func (f *fakeWalletRepository) Abort(*sqlx.Tx)                          {}
func (f *fakeWalletRepository) GetWalletForUpdate(context.Context, *sqlx.Tx, string) (service_dto.Wallet, error) {
	return service_dto.Wallet{}, errorsx.ErrWalletNotFound
}
func (f *fakeWalletRepository) UpdateBalance(context.Context, *sqlx.Tx, string, int64) error {
	return nil
}
func (f *fakeWalletRepository) FindTransactionByReference(context.Context, *sqlx.Tx, string, string, service_dto.TransactionType) (service_dto.Transaction, bool, error) {
	return service_dto.Transaction{}, false, nil
}
func (f *fakeWalletRepository) CreateTransaction(context.Context, *sqlx.Tx, string, service_dto.TransactionType, int64, string) (service_dto.Transaction, bool, error) {
	return service_dto.Transaction{}, false, nil
}
func (f *fakeWalletRepository) ListRecentTransactions(context.Context, string, int) ([]service_dto.Transaction, error) {
	return nil, nil
}

type noopCache struct{}

func (noopCache) GetRecent(context.Context, string) ([]service_dto.Transaction, bool, error) {
	return nil, false, nil
}
func (noopCache) SetRecent(context.Context, string, []service_dto.Transaction) error { return nil }
func (noopCache) Invalidate(context.Context, string) error                           { return nil }

func mustMarshal(t *testing.T, event *userpb.UserRegisteredEvent) []byte {
	t.Helper()
	b, err := proto.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return b
}

func TestUserRegisteredConsumer_CreatesWalletForAnyRole(t *testing.T) {
	for _, role := range []string{"user", "driver"} {
		repo := &fakeWalletRepository{}
		svc := service.NewWalletService(repo, noopCache{})
		c := NewUserRegisteredConsumer(svc)

		value := mustMarshal(t, &userpb.UserRegisteredEvent{UserId: "user-1", Role: role})
		c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

		if !repo.createCalled {
			t.Fatalf("role %q: expected CreateWallet to be called", role)
		}
		if repo.createArg != "user-1" {
			t.Errorf("role %q: got user ID %q, want %q", role, repo.createArg, "user-1")
		}
	}
}

func TestUserRegisteredConsumer_IgnoresAlreadyExists(t *testing.T) {
	repo := &fakeWalletRepository{createErr: errorsx.ErrWalletAlreadyExists}
	svc := service.NewWalletService(repo, noopCache{})
	c := NewUserRegisteredConsumer(svc)

	value := mustMarshal(t, &userpb.UserRegisteredEvent{UserId: "user-1", Role: "user"})
	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: value})

	if !repo.createCalled {
		t.Fatal("expected CreateWallet to be called")
	}
}

func TestUserRegisteredConsumer_SkipsUnparsableMessage(t *testing.T) {
	repo := &fakeWalletRepository{}
	svc := service.NewWalletService(repo, noopCache{})
	c := NewUserRegisteredConsumer(svc)

	c.handleMessage(context.Background(), &sarama.ConsumerMessage{Value: []byte("not-a-proto-message")})

	if repo.createCalled {
		t.Fatal("expected CreateWallet not to be called for an unparsable message")
	}
}
