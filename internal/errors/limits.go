package errors

import "fmt"

type LimitExceededError struct {
	Message string
	BaseErr error
}

func (e LimitExceededError) Error() string {
	template := "limit exceeded"
	if e.Message != "" {
		template = fmt.Sprintf(template+": %s", e.Message)
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %w", e.BaseErr).Error()
	}

	return template
}

func (e LimitExceededError) Unwrap() error {
	return e.BaseErr
}
