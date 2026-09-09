package errorsx

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrForbidden          = errors.New("token does not carry the required role")
)
