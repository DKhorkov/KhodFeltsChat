package errors

import "errors"

var (
	ErrInvalidChat         = errors.New("invalid chat")
	ErrUserIsNotChatMember = errors.New("user is not a chat member")
	ErrChatNotFound        = errors.New("chat not found")
)
