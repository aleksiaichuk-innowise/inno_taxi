package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/jmoiron/sqlx"
)

func (r *PgRepository) FindTransactionByReference(ctx context.Context, tx *sqlx.Tx, walletID, referenceID string, t service_dto.TransactionType) (service_dto.Transaction, bool, error) {
	const query = `
		SELECT id, wallet_id, type, amount_minor_units, reference_id, created_at
		FROM transactions
		WHERE wallet_id = $1 AND reference_id = $2 AND type = $3
	`

	var row repo_entity.Transaction
	if err := tx.GetContext(ctx, &row, query, walletID, referenceID, string(t)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service_dto.Transaction{}, false, nil
		}
		return service_dto.Transaction{}, false, fmt.Errorf("find transaction by reference: %w", err)
	}

	return transactionFromRow(row), true, nil
}

func transactionFromRow(row repo_entity.Transaction) service_dto.Transaction {
	return service_dto.Transaction{
		ID:               row.ID,
		WalletID:         row.WalletID,
		Type:             service_dto.TransactionType(row.Type),
		AmountMinorUnits: row.AmountMinorUnits,
		ReferenceID:      row.ReferenceID,
		CreatedAt:        row.CreatedAt,
	}
}
