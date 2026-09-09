package pg_repo

import (
	"context"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
)

func (r *PgRepository) ListRecentTransactions(ctx context.Context, walletID string, limit int) ([]service_dto.Transaction, error) {
	const query = `
		SELECT id, wallet_id, type, amount_minor_units, reference_id, created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	var rows []repo_entity.Transaction
	if err := r.db.SelectContext(ctx, &rows, query, walletID, limit); err != nil {
		return nil, fmt.Errorf("list recent transactions: %w", err)
	}

	txs := make([]service_dto.Transaction, 0, len(rows))
	for _, row := range rows {
		txs = append(txs, transactionFromRow(row))
	}
	return txs, nil
}
