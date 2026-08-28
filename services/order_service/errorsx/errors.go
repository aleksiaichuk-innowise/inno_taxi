package errorsx

import "errors"

var (
	ErrInvalidTaxiType = errors.New("invalid taxi type")
	ErrInvalidLocation = errors.New("invalid location")
)
