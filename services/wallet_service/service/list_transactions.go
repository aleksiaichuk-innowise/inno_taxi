package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
)

const recentTransactionsLimit = 20

func (s *WalletService) ListRecentTransactions(ctx context.Context, userID string) ([]service_dto.Transaction, error) {
	wallet, err := s.repo.GetWalletByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if cached, ok, err := s.cache.GetRecent(ctx, wallet.ID); err != nil {
		slog.ErrorContext(ctx, "read recent transactions cache", "wallet_id", wallet.ID, "error", err)
	} else if ok {
		return cached, nil
	}

	txs, err := s.repo.ListRecentTransactions(ctx, wallet.ID, recentTransactionsLimit)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetRecent(ctx, wallet.ID, txs); err != nil {
		slog.ErrorContext(ctx, "write recent transactions cache", "wallet_id", wallet.ID, "error", err)
	}

	return txs, nil
}
