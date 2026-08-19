package errorsx

import "errors"

var (
	ErrDriverNotFound      = errors.New("driver not found")
	ErrDriverAlreadyExists = errors.New("driver already exists")
	ErrInvalidTaxiType     = errors.New("invalid taxi type")
	ErrInvalidStatus       = errors.New("invalid status")
)
