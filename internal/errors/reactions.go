package errors

import "errors"

var (
	ErrReactionAlreadyExists = errors.New("reaction already exists on this message for this user")
	ErrReactionNotFound      = errors.New("reaction not found in dictionary")
	ErrReactionNotSet        = errors.New("reaction was not set on this message for this user")
)
