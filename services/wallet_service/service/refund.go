package service

import (
	"context"
	"log/slog"

	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/jmoiron/sqlx"
)

// Refund returns (transaction, created, error), same idempotent-replay
// contract as Charge.
func (s *WalletService) Refund(ctx context.Context, userID string, amountMinorUnits int64, referenceID string) (service_dto.Transaction, bool, error) {
	if amountMinorUnits <= 0 {
		slog.WarnContext(ctx, "wallet refund rejected", "user_id", userID, "reference_id", referenceID, "amount_minor_units", amountMinorUnits, "reason", errorsx.ErrInvalidAmount)
		return service_dto.Transaction{}, false, errorsx.ErrInvalidAmount
	}

	var result service_dto.Transaction
	var created bool

	err := s.transactionalOperation(ctx, func(tx *sqlx.Tx) error {
		wallet, err := s.repo.GetWalletForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		if existing, found, err := s.repo.FindTransactionByReference(ctx, tx, wallet.ID, referenceID, service_dto.TransactionTypeCredit); err != nil {
			return err
		} else if found {
			result, created = existing, false
			return nil
		}

		if err := s.repo.UpdateBalance(ctx, tx, wallet.ID, wallet.BalanceMinorUnits+amountMinorUnits); err != nil {
			return err
		}

		record, isNew, err := s.repo.CreateTransaction(ctx, tx, wallet.ID, service_dto.TransactionTypeCredit, amountMinorUnits, referenceID)
		if err != nil {
			return err
		}
		result, created = record, isNew
		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, "wallet refund rejected", "user_id", userID, "reference_id", referenceID, "amount_minor_units", amountMinorUnits, "reason", err)
		return service_dto.Transaction{}, false, err
	}

	slog.InfoContext(ctx, "wallet refund recorded", "wallet_id", result.WalletID, "reference_id", referenceID, "amount_minor_units", amountMinorUnits, "replay", !created)

	if created {
		if err := s.cache.Invalidate(ctx, result.WalletID); err != nil {
			slog.ErrorContext(ctx, "invalidate recent transactions cache", "wallet_id", result.WalletID, "error", err)
		}
	}

	return result, created, nil
}
