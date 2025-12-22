package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccessTokenDoesNotBelongToRefreshTokenError(t *testing.T) {
	testCases := []struct {
		name           string
		err            AccessTokenDoesNotBelongToRefreshTokenError
		expectedString string
		expectedBase   error
	}{
		{
			name:           "default message, no base error",
			err:            AccessTokenDoesNotBelongToRefreshTokenError{},
			expectedString: "access token does not belong to refresh token",
			expectedBase:   nil,
		},
		{
			name:           "custom message, no base error",
			err:            AccessTokenDoesNotBelongToRefreshTokenError{Message: "invalid token pair"},
			expectedString: "invalid token pair",
			expectedBase:   nil,
		},
		{
			name:           "default message, with base error",
			err:            AccessTokenDoesNotBelongToRefreshTokenError{BaseErr: errors.New("token mismatch")},
			expectedString: "access token does not belong to refresh token. Base error: token mismatch",
			expectedBase:   errors.New("token mismatch"),
		},
		{
			name:           "custom message, with base error",
			err:            AccessTokenDoesNotBelongToRefreshTokenError{Message: "invalid token pair", BaseErr: errors.New("token mismatch")},
			expectedString: "invalid token pair. Base error: token mismatch",
			expectedBase:   errors.New("token mismatch"),
		},
		{
			name:           "empty message, with base error",
			err:            AccessTokenDoesNotBelongToRefreshTokenError{Message: "", BaseErr: errors.New("token mismatch")},
			expectedString: "access token does not belong to refresh token. Base error: token mismatch",
			expectedBase:   errors.New("token mismatch"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Проверка строки ошибки
			require.Equal(t, tc.expectedString, tc.err.Error())

			// Проверка базовой ошибки через Unwrap
			baseErr := tc.err.Unwrap()
			if tc.expectedBase == nil {
				require.Nil(t, baseErr)
			} else {
				require.Equal(t, tc.expectedBase.Error(), baseErr.Error())
			}

			// Проверка, что ошибка реализует интерфейс error
			var err interface{} = tc.err
			_, ok := err.(error)
			require.True(t, ok, "AccessTokenDoesNotBelongToRefreshTokenError should implement error interface")
		})
	}
}
