package errors

import "fmt"

var (
	ErrEmailNotConfirmed                      = fmt.Errorf("email not confirmed")
	ErrEmailAlreadyConfirmed                  = fmt.Errorf("email already confirmed")
	ErrWrongPassword                          = fmt.Errorf("wrong password")
	ErrAccessTokenDoesNotBelongToRefreshToken = fmt.Errorf("access token does not belong to refresh token")
)
