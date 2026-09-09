package service

import (
	"context"
	"sync"
	"time"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/jmoiron/sqlx"
)

// fakeWalletRepository is an in-memory stand-in good enough to exercise the
// service layer's own logic (validation, idempotency branching, cache
// invalidation calls) without a real Postgres transaction underneath -
// Start/Finish/Abort are no-ops here, and every "tx-scoped" method just
// ignores the *sqlx.Tx it's handed. The things a fake fundamentally can't
// prove (the FOR UPDATE lock, the unique constraint) are covered instead
// by repository/pg_repo's dockertest suite.
type fakeWalletRepository struct {
	mu      sync.Mutex
	wallets map[string]service_dto.Wallet // keyed by user_id
	byID    map[string]string             // wallet_id -> user_id
	txs     map[string][]service_dto.Transaction

	createWalletErr error
}

func newFakeWalletRepository() *fakeWalletRepository {
	return &fakeWalletRepository{
		wallets: map[string]service_dto.Wallet{},
		byID:    map[string]string{},
		txs:     map[string][]service_dto.Transaction{},
	}
}

func (f *fakeWalletRepository) CreateWallet(_ context.Context, userID string) (service_dto.Wallet, error) {
	if f.createWalletErr != nil {
		return service_dto.Wallet{}, f.createWalletErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.wallets[userID]; ok {
		return service_dto.Wallet{}, errorsx.ErrWalletAlreadyExists
	}
	w := service_dto.Wallet{ID: "wallet-" + userID, UserID: userID}
	f.wallets[userID] = w
	f.byID[w.ID] = userID
	return w, nil
}

func (f *fakeWalletRepository) GetWalletByUserID(_ context.Context, userID string) (service_dto.Wallet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.wallets[userID]
	if !ok {
		return service_dto.Wallet{}, errorsx.ErrWalletNotFound
	}
	return w, nil
}

func (f *fakeWalletRepository) Start(_ context.Context) (*sqlx.Tx, error) { return nil, nil }
func (f *fakeWalletRepository) Finish(_ *sqlx.Tx) error                   { return nil }
func (f *fakeWalletRepository) Abort(_ *sqlx.Tx)                          {}

func (f *fakeWalletRepository) GetWalletForUpdate(_ context.Context, _ *sqlx.Tx, userID string) (service_dto.Wallet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.wallets[userID]
	if !ok {
		return service_dto.Wallet{}, errorsx.ErrWalletNotFound
	}
	return w, nil
}

func (f *fakeWalletRepository) UpdateBalance(_ context.Context, _ *sqlx.Tx, walletID string, newBalance int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID := f.byID[walletID]
	w := f.wallets[userID]
	w.BalanceMinorUnits = newBalance
	f.wallets[userID] = w
	return nil
}

func (f *fakeWalletRepository) FindTransactionByReference(_ context.Context, _ *sqlx.Tx, walletID, referenceID string, t service_dto.TransactionType) (service_dto.Transaction, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, tx := range f.txs[walletID] {
		if tx.ReferenceID == referenceID && tx.Type == t {
			return tx, true, nil
		}
	}
	return service_dto.Transaction{}, false, nil
}

func (f *fakeWalletRepository) CreateTransaction(_ context.Context, _ *sqlx.Tx, walletID string, t service_dto.TransactionType, amount int64, referenceID string) (service_dto.Transaction, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tx := service_dto.Transaction{
		ID:               "tx-" + referenceID,
		WalletID:         walletID,
		Type:             t,
		AmountMinorUnits: amount,
		ReferenceID:      referenceID,
		CreatedAt:        time.Now(),
	}
	f.txs[walletID] = append(f.txs[walletID], tx)
	return tx, true, nil
}

func (f *fakeWalletRepository) ListRecentTransactions(_ context.Context, walletID string, limit int) ([]service_dto.Transaction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	txs := f.txs[walletID]
	if len(txs) > limit {
		txs = txs[len(txs)-limit:]
	}
	return txs, nil
}

type fakeTransactionCache struct {
	mu             sync.Mutex
	cached         map[string][]service_dto.Transaction
	invalidateCall map[string]int
}

func newFakeTransactionCache() *fakeTransactionCache {
	return &fakeTransactionCache{
		cached:         map[string][]service_dto.Transaction{},
		invalidateCall: map[string]int{},
	}
}

func (c *fakeTransactionCache) GetRecent(_ context.Context, walletID string) ([]service_dto.Transaction, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	txs, ok := c.cached[walletID]
	return txs, ok, nil
}

func (c *fakeTransactionCache) SetRecent(_ context.Context, walletID string, txs []service_dto.Transaction) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached[walletID] = txs
	return nil
}

func (c *fakeTransactionCache) Invalidate(_ context.Context, walletID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cached, walletID)
	c.invalidateCall[walletID]++
	return nil
}
