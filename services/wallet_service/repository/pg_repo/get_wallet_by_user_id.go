package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
)

func (r *PgRepository) GetWalletByUserID(ctx context.Context, userID string) (service_dto.Wallet, error) {
	const query = `
		SELECT id, user_id, balance_minor_units, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
	`

	var row repo_entity.Wallet
	if err := r.db.GetContext(ctx, &row, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service_dto.Wallet{}, errorsx.ErrWalletNotFound
		}
		return service_dto.Wallet{}, fmt.Errorf("get wallet by user id: %w", err)
	}

	return walletFromRow(row), nil
}
