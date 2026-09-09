package errorsx

import "errors"

var (
	ErrWalletAlreadyExists = errors.New("wallet already exists")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrInsufficientFunds   = errors.New("insufficient funds")
)
