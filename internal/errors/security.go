package errors

import "errors"

var (
	ErrInvalidJWT = errors.New("invalid jwt token")
)
