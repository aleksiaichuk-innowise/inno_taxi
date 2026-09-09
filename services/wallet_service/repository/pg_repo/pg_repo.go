package pg_repo

import "github.com/jmoiron/sqlx"

type PgRepository struct {
	db *sqlx.DB
}

func NewPgRepo(db *sqlx.DB) *PgRepository {
	return &PgRepository{db: db}
}
