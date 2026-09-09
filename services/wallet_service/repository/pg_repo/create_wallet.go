package pg_repo

import (
	"context"
	"errors"
	"fmt"

	repo_entity "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/repository"
	service_dto "github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/entity/service"
	"github.com/aleksiaichuk-innowise/inno_taxi/services/wallet_service/errorsx"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PgRepository) CreateWallet(ctx context.Context, userID string) (service_dto.Wallet, error) {
	const query = `
		INSERT INTO wallets (user_id)
		VALUES ($1)
		RETURNING id, user_id, balance_minor_units, created_at, updated_at
	`

	var row repo_entity.Wallet
	if err := r.db.GetContext(ctx, &row, query, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return service_dto.Wallet{}, errorsx.ErrWalletAlreadyExists
		}
		return service_dto.Wallet{}, fmt.Errorf("create wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func walletFromRow(row repo_entity.Wallet) service_dto.Wallet {
	return service_dto.Wallet{
		ID:                row.ID,
		UserID:            row.UserID,
		BalanceMinorUnits: row.BalanceMinorUnits,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
