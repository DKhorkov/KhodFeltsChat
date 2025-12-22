package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeoutExceededError(t *testing.T) {
	testCases := []struct {
		name           string
		err            TimeoutExceededError
		expectedString string
		expectedBase   error
	}{
		{
			name: "default message without base error",
			err: TimeoutExceededError{
				Message: "",
				BaseErr: nil,
			},
			expectedString: "timeout exceeded",
			expectedBase:   nil,
		},
		{
			name: "default message with base error",
			err: TimeoutExceededError{
				Message: "",
				BaseErr: errors.New("too many tags"),
			},
			expectedString: "timeout exceeded. Base error: too many tags",
			expectedBase:   errors.New("too many tags"),
		},
		{
			name: "custom message without base error",
			err: TimeoutExceededError{
				Message: "too many tags",
				BaseErr: nil,
			},
			expectedString: "timeout exceeded: too many tags",
			expectedBase:   nil,
		},
		{
			name: "custom message with base error",
			err: TimeoutExceededError{
				Message: "too many tags",
				BaseErr: errors.New("too many tags"),
			},
			expectedString: "timeout exceeded: too many tags. Base error: too many tags",
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
