package errors

import "fmt"

type UserNotFoundError struct {
	Message string
	BaseErr error
}

func (e UserNotFoundError) Error() string {
	template := "user not found"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e UserNotFoundError) Unwrap() error {
	return e.BaseErr
}
