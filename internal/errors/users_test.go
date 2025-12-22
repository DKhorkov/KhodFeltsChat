package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserNotFoundError(t *testing.T) {
	testCases := []struct {
		name           string
		err            UserNotFoundError
		expectedString string
		expectedBase   error
	}{
		{
			name:           "default message, no base error",
			err:            UserNotFoundError{},
			expectedString: "user not found",
			expectedBase:   nil,
		},
		{
			name:           "custom message, no base error",
			err:            UserNotFoundError{Message: "user with ID=1 not found"},
			expectedString: "user with ID=1 not found",
			expectedBase:   nil,
		},
		{
			name:           "default message, with base error",
			err:            UserNotFoundError{BaseErr: errors.New("database error")},
			expectedString: "user not found. Base error: database error",
			expectedBase:   errors.New("database error"),
		},
		{
			name:           "custom message, with base error",
			err:            UserNotFoundError{Message: "user with ID=1 not found", BaseErr: errors.New("database error")},
			expectedString: "user with ID=1 not found. Base error: database error",
			expectedBase:   errors.New("database error"),
		},
		{
			name:           "empty message, with base error",
			err:            UserNotFoundError{Message: "", BaseErr: errors.New("database error")},
			expectedString: "user not found. Base error: database error",
			expectedBase:   errors.New("database error"),
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
			require.True(t, ok, "UserNotFoundError should implement error interface")
		})
	}
}
