package pg_repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func (r *PgRepository) UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID string, newBalance int64) error {
	const query = `
		UPDATE wallets
		SET balance_minor_units = $1, updated_at = now()
		WHERE id = $2
	`

	if _, err := tx.ExecContext(ctx, query, newBalance, walletID); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	return nil
}
