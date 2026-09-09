package pg_repo

import (
	"context"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// CreateTransaction inserts the transaction row. The returned bool is false
// only when a concurrent request won the race on the same
// (wallet_id, reference_id, type) - the unique constraint rejected this
// insert, and the row returned is the winner's, not a new one. Callers
// that already checked FindTransactionByReference before calling this
// still need to handle that bool: the check-then-insert isn't atomic on
// its own, only the constraint is.
func (r *PgRepository) CreateTransaction(ctx context.Context, tx *sqlx.Tx, walletID string, t service_dto.TransactionType, amount int64, referenceID string) (service_dto.Transaction, bool, error) {
	const insertQuery = `
		INSERT INTO transactions (wallet_id, type, amount_minor_units, reference_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, wallet_id, type, amount_minor_units, reference_id, created_at
	`

	var row repo_entity.Transaction
	err := tx.GetContext(ctx, &row, insertQuery, walletID, string(t), amount, referenceID)
	if err == nil {
		return transactionFromRow(row), true, nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return service_dto.Transaction{}, false, fmt.Errorf("create transaction: %w", err)
	}

	existing, found, findErr := r.FindTransactionByReference(ctx, tx, walletID, referenceID, t)
	if findErr != nil {
		return service_dto.Transaction{}, false, fmt.Errorf("create transaction: lost the unique-constraint race and couldn't refetch it: %w", findErr)
	}
	if !found {
		return service_dto.Transaction{}, false, fmt.Errorf("create transaction: unique constraint violated but no matching row found")
	}
	return existing, false, nil
}
