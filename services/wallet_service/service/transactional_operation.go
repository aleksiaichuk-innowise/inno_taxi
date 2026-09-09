package service

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// transactionalOperation wraps fn in a Postgres transaction: Start before,
// Finish (COMMIT) after fn returns nil, Abort (ROLLBACK, via defer) on any
// error - including one that happens after Finish has already committed,
// where Abort's Rollback call is a harmless no-op.
func (s *WalletService) transactionalOperation(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.repo.Start(ctx)
	if err != nil {
		return err
	}
	defer s.repo.Abort(tx)

	if err := fn(tx); err != nil {
		return err
	}
	return s.repo.Finish(tx)
}
