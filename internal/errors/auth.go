package errors

import "errors"

var (
	ErrEmailNotConfirmed                      = errors.New("email not confirmed")
	ErrEmailAlreadyConfirmed                  = errors.New("email already confirmed")
	ErrWrongPassword                          = errors.New("wrong password")
	ErrAccessTokenDoesNotBelongToRefreshToken = errors.New(
		"access token does not belong to refresh token",
	)
	ErrNewPasswordEqualToOldPassword = errors.New(`new password equal to old password`)
)
