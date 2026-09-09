package repository

import "time"

type Transaction struct {
	ID               string    `db:"id"`
	WalletID         string    `db:"wallet_id"`
	Type             string    `db:"type"`
	AmountMinorUnits int64     `db:"amount_minor_units"`
	ReferenceID      string    `db:"reference_id"`
	CreatedAt        time.Time `db:"created_at"`
}
