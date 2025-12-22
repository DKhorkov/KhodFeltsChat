package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitExceededError(t *testing.T) {
	testCases := []struct {
		name           string
		err            LimitExceededError
		expectedString string
		expectedBase   error
	}{
		{
			name: "default message without base error",
			err: LimitExceededError{
				Message: "",
				BaseErr: nil,
			},
			expectedString: "limit exceeded",
			expectedBase:   nil,
		},
		{
			name: "default message with base error",
			err: LimitExceededError{
				Message: "",
				BaseErr: errors.New("too many tags"),
			},
			expectedString: "limit exceeded. Base error: too many tags",
			expectedBase:   errors.New("too many tags"),
		},
		{
			name: "custom message without base error",
			err: LimitExceededError{
				Message: "too many tags",
				BaseErr: nil,
			},
			expectedString: "limit exceeded: too many tags",
			expectedBase:   nil,
		},
		{
			name: "custom message with base error",
			err: LimitExceededError{
				Message: "too many tags",
				BaseErr: errors.New("too many tags"),
			},
			expectedString: "limit exceeded: too many tags. Base error: too many tags",
			expectedBase:   errors.New("too many tags"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualString := tc.err.Error()
			actualBase := tc.err.Unwrap()

			require.Equal(t, tc.expectedString, actualString)
			require.Equal(t, tc.expectedBase, actualBase)
		})
	}
}
