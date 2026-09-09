package pg_repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type PgAtomicRepository interface {
	Start(ctx context.Context) (*sqlx.Tx, error)
	Finish(tx *sqlx.Tx) error
	Abort(tx *sqlx.Tx)
}

func (r *PgRepository) Start(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *PgRepository) Finish(tx *sqlx.Tx) error {
	return tx.Commit()
}

func (r *PgRepository) Abort(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("rollback transaction", "error", err)
	}
}
