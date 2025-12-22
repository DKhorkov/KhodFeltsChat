package errors

import "fmt"

type TimeoutExceededError struct {
	Message string
	BaseErr error
}

func (e TimeoutExceededError) Error() string {
	template := "timeout exceeded"
	if e.Message != "" {
		template = fmt.Sprintf(template+": %s", e.Message)
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %w", e.BaseErr).Error()
	}

	return template
}

func (e TimeoutExceededError) Unwrap() error {
	return e.BaseErr
}
