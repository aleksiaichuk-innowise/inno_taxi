package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/jmoiron/sqlx"
)

// GetWalletForUpdate locks the wallet row for the lifetime of tx so two
// concurrent charge/refund calls against the same wallet can't both read
// the same starting balance.
func (r *PgRepository) GetWalletForUpdate(ctx context.Context, tx *sqlx.Tx, userID string) (service_dto.Wallet, error) {
	const query = `
		SELECT id, user_id, balance_minor_units, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
		FOR UPDATE
	`

	var row repo_entity.Wallet
	if err := tx.GetContext(ctx, &row, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service_dto.Wallet{}, errorsx.ErrWalletNotFound
		}
		return service_dto.Wallet{}, fmt.Errorf("get wallet for update: %w", err)
	}

	return walletFromRow(row), nil
}
