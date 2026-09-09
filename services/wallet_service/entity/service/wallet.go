package service

import "time"

type Wallet struct {
	ID                string
	UserID            string
	BalanceMinorUnits int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TransactionType string

const (
	TransactionTypeDebit  TransactionType = "debit"
	TransactionTypeCredit TransactionType = "credit"
)

type Transaction struct {
	ID               string
	WalletID         string
	Type             TransactionType
	AmountMinorUnits int64
	ReferenceID      string
	CreatedAt        time.Time
}
