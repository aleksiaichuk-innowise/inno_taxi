package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/jmoiron/sqlx"
)

// Charge returns (transaction, created, error). created is false when this
// call is an idempotent replay of an already-processed reference_id - the
// caller (HTTP handler) uses it to choose between 201 and 200.
func (s *WalletService) Charge(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) (service_dto.Transaction, bool, error) {
	if amountMinorUnits <= 0 {
		return service_dto.Transaction{}, false, errorsx.ErrInvalidAmount
	}

	var result service_dto.Transaction
	var created bool

	err := s.transactionalOperation(ctx, func(tx *sqlx.Tx) error {
		wallet, err := s.repo.GetWalletForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		if existing, found, err := s.repo.FindTransactionByReference(ctx, tx, wallet.ID, referenceID, service_dto.TransactionTypeDebit); err != nil {
			return err
		} else if found {
			result, created = existing, false
			return nil
		}

		if wallet.BalanceMinorUnits < amountMinorUnits {
			return errorsx.ErrInsufficientFunds
		}

		if err := s.repo.UpdateBalance(ctx, tx, wallet.ID, wallet.BalanceMinorUnits-amountMinorUnits); err != nil {
			return err
		}

		record, isNew, err := s.repo.CreateTransaction(ctx, tx, wallet.ID, service_dto.TransactionTypeDebit, amountMinorUnits, referenceID)
		if err != nil {
			return err
		}
		result, created = record, isNew
		return nil
	})
	if err != nil {
		return service_dto.Transaction{}, false, err
	}

	if created {
		if err := s.cache.Invalidate(ctx, result.WalletID); err != nil {
			slog.ErrorContext(ctx, "invalidate recent transactions cache", "wallet_id", result.WalletID, "error", err)
		}
	}

	return result, created, nil
}
