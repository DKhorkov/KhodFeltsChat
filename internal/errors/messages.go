package errors

import "errors"

var (
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageAuthor = errors.New("only message author can delete for all")
)
