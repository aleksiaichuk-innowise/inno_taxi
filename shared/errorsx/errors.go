package errorsx

import "errors"

var (
	ErrPaymentNotFound = errors.New("payment not found")
	ErrInternal        = errors.New("internal server error")
)

type HttpErrResp struct {
	Message string `json:"message"`
}
