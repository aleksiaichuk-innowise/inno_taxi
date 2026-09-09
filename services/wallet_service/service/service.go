package service

import (
	"context"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/jmoiron/sqlx"
)

type WalletRepository interface {
	CreateWallet(ctx context.Context, userID string) (service_dto.Wallet, error)
	GetWalletByUserID(ctx context.Context, userID string) (service_dto.Wallet, error)

	Start(ctx context.Context) (*sqlx.Tx, error)
	Finish(tx *sqlx.Tx) error
	Abort(tx *sqlx.Tx)

	GetWalletForUpdate(ctx context.Context, tx *sqlx.Tx, userID string) (service_dto.Wallet, error)
	UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID string, newBalance int64) error
	FindTransactionByReference(ctx context.Context, tx *sqlx.Tx, walletID, referenceID string, t service_dto.TransactionType) (service_dto.Transaction, bool, error)
	CreateTransaction(ctx context.Context, tx *sqlx.Tx, walletID string, t service_dto.TransactionType, amount int64, referenceID string) (service_dto.Transaction, bool, error)

	ListRecentTransactions(ctx context.Context, walletID string, limit int) ([]service_dto.Transaction, error)
}

type TransactionCache interface {
	GetRecent(ctx context.Context, walletID string) ([]service_dto.Transaction, bool, error)
	SetRecent(ctx context.Context, walletID string, txs []service_dto.Transaction) error
	Invalidate(ctx context.Context, walletID string) error
}

type WalletService struct {
	repo  WalletRepository
	cache TransactionCache
}

func NewWalletService(repo WalletRepository, cache TransactionCache) *WalletService {
	return &WalletService{repo: repo, cache: cache}
}
