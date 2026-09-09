package repository

import "time"

type Wallet struct {
	ID                string    `db:"id"`
	UserID            string    `db:"user_id"`
	BalanceMinorUnits int64     `db:"balance_minor_units"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
