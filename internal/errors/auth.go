package errors

import "fmt"

type WrongPasswordError struct {
	Message string
	BaseErr error
}

func (e WrongPasswordError) Error() string {
	template := "wrong password"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e WrongPasswordError) Unwrap() error {
	return e.BaseErr
}

type UserAlreadyExistsError struct {
	Message string
	BaseErr error
}

func (e UserAlreadyExistsError) Error() string {
	template := "user with provided email already exists"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e UserAlreadyExistsError) Unwrap() error {
	return e.BaseErr
}

type RefreshTokenAlreadyExistsError struct {
	Message string
	BaseErr error
}

func (e RefreshTokenAlreadyExistsError) Error() string {
	template := "refresh token already exists"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e RefreshTokenAlreadyExistsError) Unwrap() error {
	return e.BaseErr
}

type RefreshTokenNotFoundError struct {
	Message string
	BaseErr error
}

func (e RefreshTokenNotFoundError) Error() string {
	template := "refresh token not found"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e RefreshTokenNotFoundError) Unwrap() error {
	return e.BaseErr
}

type EmailAlreadyConfirmedError struct {
	Message string
	BaseErr error
}

func (e EmailAlreadyConfirmedError) Error() string {
	template := "provided email has been already confirmed"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e EmailAlreadyConfirmedError) Unwrap() error {
	return e.BaseErr
}

type EmailIsNotConfirmedError struct {
	Message string
	BaseErr error
}

func (e EmailIsNotConfirmedError) Error() string {
	template := "provided email is not confirmed"
	if e.Message != "" {
		template = e.Message
	}

	if e.BaseErr != nil {
		return fmt.Errorf(template+". Base error: %v", e.BaseErr).Error()
	}

	return template
}

func (e EmailIsNotConfirmedError) Unwrap() error {
	return e.BaseErr
}
