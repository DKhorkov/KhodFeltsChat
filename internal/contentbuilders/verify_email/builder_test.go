package verify_email_test

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	"github.com/DKhorkov/kfc/internal/domains"
	cachemocks "github.com/DKhorkov/libs/cache/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestContentBuilder_Subject(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	builder := verify_email.New("http://example.com/verify-email", mockCache)

	require.Equal(t, "Подтверждение адреса электронной почты", builder.Subject())
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

	linkRegexp := regexp.MustCompile(`http://example\.com/verify-email/(\d{6})`)

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

			builder := verify_email.New("http://example.com/verify-email", mockCache)

			result, err := builder.Body(context.Background(), tc.user)
			require.NoError(t, err)
			require.Contains(t, result, tc.user.Username)

			linkMatches := linkRegexp.FindStringSubmatch(result)
			require.Len(t, linkMatches, 2, "should contain 6-digit code in link")

			code, err := strconv.ParseUint(linkMatches[1], 10, 64)
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

	builder := verify_email.New("http://example.com/verify-email", mockCache)

	_, err := builder.Body(context.Background(), domains.User{ID: 1, Username: "Alice"})
	require.Error(t, err)
}

func TestContentBuilder_Body_Collision(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	mockCache.EXPECT().
		SetNX(gomock.Any(), gomock.Any(), gomock.Any(), common.TokenTTL).
		Return(nil).
		Times(common.OTPGenerateAttempts)
	mockCache.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return("other-user", nil).
		Times(common.OTPGenerateAttempts)

	builder := verify_email.New("http://example.com/verify-email", mockCache)

	_, err := builder.Body(context.Background(), domains.User{ID: 1, Username: "Alice"})
	require.ErrorIs(t, err, common.ErrOTPCollision)
}
