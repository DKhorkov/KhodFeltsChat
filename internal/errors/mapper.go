package errors

import (
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"-"`
}

var (
	ErrValidationFailed                       = errors.New("validation failed")
	ErrUserAlreadyExists                      = errors.New("user already exists")
	ErrUserNotFound                           = errors.New("user not found")
	ErrInvalidJWT                             = errors.New("invalid jwt token")
	ErrInvalidChat                            = errors.New("invalid chat")
	ErrUserIsNotChatMember                    = errors.New("user is not a chat member")
	ErrChatNotFound                           = errors.New("chat not found")
	ErrChatAlreadyExists                      = errors.New("chat already exists")
	ErrEmailNotConfirmed                      = errors.New("email not confirmed")
	ErrEmailAlreadyConfirmed                  = errors.New("email already confirmed")
	ErrWrongPassword                          = errors.New("wrong password")
	ErrAccessTokenDoesNotBelongToRefreshToken = errors.New(
		"access token does not belong to refresh token",
	)
)

type ErrorMapper struct{}

var DefaultErrorMapper = ErrorMapper{}

func (m *ErrorMapper) Map(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{
			Status:  http.StatusOK,
			Message: "",
		}
	}

	switch {
	case errors.Is(err, ErrUserAlreadyExists):
		return ErrorResponse{
			Status:  http.StatusConflict,
			Message: ErrUserAlreadyExists.Error(),
		}
	case errors.Is(err, ErrUserNotFound):
		return ErrorResponse{
			Status:  http.StatusNotFound,
			Message: ErrUserNotFound.Error(),
		}
	case errors.Is(err, ErrInvalidJWT):
		return ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: ErrInvalidJWT.Error(),
		}
	case errors.Is(err, ErrInvalidChat):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrInvalidChat.Error(),
		}
	case errors.Is(err, ErrChatAlreadyExists):
		return ErrorResponse{
			Status:  http.StatusConflict,
			Message: ErrChatAlreadyExists.Error(),
		}
	case errors.Is(err, ErrEmailNotConfirmed):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrEmailNotConfirmed.Error(),
		}
	case errors.Is(err, ErrEmailAlreadyConfirmed):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrEmailAlreadyConfirmed.Error(),
		}
	case errors.Is(err, ErrWrongPassword):
		return ErrorResponse{
			Status:  http.StatusUnauthorized,
			Message: ErrWrongPassword.Error(),
		}
	case errors.Is(err, ErrAccessTokenDoesNotBelongToRefreshToken):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrAccessTokenDoesNotBelongToRefreshToken.Error(),
		}
	case errors.Is(err, ErrUserIsNotChatMember):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrUserIsNotChatMember.Error(),
		}
	case errors.Is(err, ErrChatNotFound):
		return ErrorResponse{
			Status:  http.StatusNotFound,
			Message: ErrChatNotFound.Error(),
		}
	case errors.Is(err, ErrValidationFailed):
		return ErrorResponse{
			Status:  http.StatusBadRequest,
			Message: ErrValidationFailed.Error(),
		}
	default:
		return ErrorResponse{
			Status:  http.StatusInternalServerError,
			Message: "internal server error",
		}
	}
}
