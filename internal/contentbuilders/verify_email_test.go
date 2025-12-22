package contentbuilders

import (
	"github.com/DKhorkov/khodfeltschat/internal/domains"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerifyEmailContentBuilder_Subject(t *testing.T) {
	builder := NewVerifyEmailContentBuilder("http://example.com/verify-email")

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
			result := builder.Subject()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestVerifyEmailContentBuilder_Body(t *testing.T) {
	builder := NewVerifyEmailContentBuilder("http://example.com/verify-email")

	testCases := []struct {
		name     string
		user     domains.User
		expected string
	}{
		{
			name: "basic user",
			user: domains.User{
				ID:       1,
				Username: "Alice",
			},
			expected: `<p>Добрый день, Alice!</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/verify-email/MQ">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`,
		},
		{
			name: "user with special characters",
			user: domains.User{
				ID:       123,
				Username: "Bob <Test>",
			},
			expected: `<p>Добрый день, Bob <Test>!</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/verify-email/MTIz">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`,
		},
		{
			name: "user with large ID",
			user: domains.User{
				ID:       987654321,
				Username: "Charlie",
			},
			expected: `<p>Добрый день, Charlie!</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/verify-email/OTg3NjU0MzIx">ссылке</a>, чтобы подтвердить адрес электронной почты!</p>
<p>С уважением,<br>
команда Handmade Toys Marketplace.</p>
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.Body(tc.user)
			require.Equal(t, tc.expected, result)
		})
	}
}
