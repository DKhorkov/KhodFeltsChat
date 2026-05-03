package verify_email_test

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	"github.com/DKhorkov/kfc/internal/domains"
	cachemocks "github.com/DKhorkov/libs/cache/mocks"
	"github.com/DKhorkov/libs/security"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestContentBuilder_Subject(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	builder := verify_email.New("http://example.com/verify-email", mockCache)

	testCases := []struct {
		name     string
		expected string
	}{
		{
			name:     "default subject",
			expected: "Подтверждение адреса электронной почты",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := builder.Subject()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestContentBuilder_Body(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		user domains.User
	}{
		{
			name: "basic user",
			user: domains.User{
				ID:       1,
				Username: "Alice",
			},
		},
		{
			name: "user with special characters",
			user: domains.User{
				ID:       123,
				Username: "Bob <Test>",
			},
		},
		{
			name: "user with large ID",
			user: domains.User{
				ID:       987654321,
				Username: "Charlie",
			},
		},
	}

	linkRegexp := regexp.MustCompile(`http://example\.com/verify-email/([A-Za-z0-9_-]+)`)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockCache := cachemocks.NewMockProvider(ctrl)

			mockCache.EXPECT().
				Set(gomock.Any(), gomock.Any(), gomock.Any(), common.TokenTTL).
				Return(nil).
				Times(2)

			builder := verify_email.New("http://example.com/verify-email", mockCache)

			result, err := builder.Body(context.Background(), tc.user)
			require.NoError(t, err)
			require.Contains(t, result, tc.user.Username)

			matches := linkRegexp.FindStringSubmatch(result)
			require.Len(t, matches, 2, "should contain encoded token in link")

			decoded, err := security.RawDecode(matches[1])
			require.NoError(t, err)

			_, rawUserID, found := strings.Cut(string(decoded), common.SaltSeparator)
			require.True(t, found, "decoded token should contain salt separator")
			require.Equal(t, strconv.FormatUint(tc.user.ID, 10), rawUserID)

			// Token should be different on each call (random salt).
			result2, err := builder.Body(context.Background(), tc.user)
			require.NoError(t, err)

			matches2 := linkRegexp.FindStringSubmatch(result2)
			require.Len(t, matches2, 2)
			require.NotEqual(t, matches[1], matches2[1])
		})
	}
}

func TestContentBuilder_Body_CacheError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockCache := cachemocks.NewMockProvider(ctrl)

	mockCache.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), common.TokenTTL).
		Return(errors.New("connection refused"))

	builder := verify_email.New("http://example.com/verify-email", mockCache)

	_, err := builder.Body(context.Background(), domains.User{ID: 1, Username: "Alice"})
	require.Error(t, err)
}
