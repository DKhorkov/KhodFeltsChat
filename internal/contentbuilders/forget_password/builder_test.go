package forget_password_test

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/contentbuilders/forget_password"
	"github.com/DKhorkov/kfc/internal/domains"
	cachemocks "github.com/DKhorkov/libs/cache/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestContentBuilder_Subject(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	builder := forget_password.New(mockCache)

	require.Equal(t, "Восстановление пароля от аккаунта", builder.Subject())
}

func TestContentBuilder_Body(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		user domains.User
	}{
		{
			name: "basic user",
			user: domains.User{ID: 1, Username: "Alice"},
		},
		{
			name: "user with special characters",
			user: domains.User{ID: 123, Username: "Bob <Test>"},
		},
		{
			name: "user with large ID",
			user: domains.User{ID: 987654321, Username: "Charlie"},
		},
	}

	codeRegexp := regexp.MustCompile(`<b>(\d{6})</b>`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockCache := cachemocks.NewMockProvider(ctrl)

			userIDStr := strconv.FormatUint(tc.user.ID, 10)

			mockCache.EXPECT().
				SetNX(gomock.Any(), gomock.Any(), userIDStr, common.TokenTTL).
				Return(nil).
				AnyTimes()
			mockCache.EXPECT().
				Get(gomock.Any(), gomock.Any()).
				Return(userIDStr, nil).
				AnyTimes()

			builder := forget_password.New(mockCache)

			result, err := builder.Body(context.Background(), tc.user)
			require.NoError(t, err)
			require.Contains(t, result, tc.user.Username)

			matches := codeRegexp.FindStringSubmatch(result)
			require.Len(t, matches, 2, "should contain 6-digit code in bold tag")

			code, err := strconv.ParseUint(matches[1], 10, 64)
			require.NoError(t, err)
			require.GreaterOrEqual(t, code, common.OTPMin)
			require.LessOrEqual(t, code, common.OTPMax)
		})
	}
}

func TestContentBuilder_Body_CacheSetError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	mockCache.EXPECT().
		SetNX(gomock.Any(), gomock.Any(), gomock.Any(), common.TokenTTL).
		Return(errors.New("connection refused"))

	builder := forget_password.New(mockCache)

	_, err := builder.Body(context.Background(), domains.User{ID: 1, Username: "Alice"})
	require.Error(t, err)
}

func TestContentBuilder_Body_Collision(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	// SetNX always succeeds (no error), but Get returns a different userID —
	// simulating another user already owning every generated code. Builder
	// must retry OTPGenerateAttempts times then fail with ErrOTPCollision.
	mockCache.EXPECT().
		SetNX(gomock.Any(), gomock.Any(), gomock.Any(), common.TokenTTL).
		Return(nil).
		Times(common.OTPGenerateAttempts)
	mockCache.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return("other-user", nil).
		Times(common.OTPGenerateAttempts)

	builder := forget_password.New(mockCache)

	_, err := builder.Body(context.Background(), domains.User{ID: 1, Username: "Alice"})
	require.ErrorIs(t, err, common.ErrOTPCollision)
}
