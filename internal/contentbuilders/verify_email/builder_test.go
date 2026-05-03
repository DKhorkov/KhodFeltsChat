package verify_email_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/security"
	"github.com/stretchr/testify/require"
)

func TestContentBuilder_Subject(t *testing.T) {
	t.Parallel()

	builder := verify_email.New("http://example.com/verify-email")

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

	builder := verify_email.New("http://example.com/verify-email")

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

			result := builder.Body(tc.user)

			require.Contains(t, result, tc.user.Username)

			matches := linkRegexp.FindStringSubmatch(result)
			require.Len(t, matches, 2, "should contain encoded token in link")

			decoded, err := security.RawDecode(matches[1])
			require.NoError(t, err)

			_, rawUserID, found := strings.Cut(string(decoded), common.SaltSeparator)
			require.True(t, found, "decoded token should contain salt separator")
			require.Equal(t, strconv.FormatUint(tc.user.ID, 10), rawUserID)

			// Token should be different on each call (random salt).
			result2 := builder.Body(tc.user)
			matches2 := linkRegexp.FindStringSubmatch(result2)
			require.Len(t, matches2, 2)
			require.NotEqual(t, matches[1], matches2[1])
		})
	}
}
