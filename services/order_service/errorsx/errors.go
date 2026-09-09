package errorsx

import "errors"

var (
	ErrInvalidTaxiType     = errors.New("invalid taxi type")
	ErrInvalidLocation     = errors.New("invalid location")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNotCancellable = errors.New("order is not cancellable")
	ErrOrderNotStartable   = errors.New("order is not startable")
	ErrOrderNotCompletable = errors.New("order is not completable")
)
