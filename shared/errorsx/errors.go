package errorsx

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrInternal           = errors.New("internal server error")
)

type HttpErrResp struct {
	Message string `json:"message"`
}
