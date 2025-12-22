package contentbuilders

import (
	"github.com/DKhorkov/khodfeltschat/internal/domains"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForgetPasswordContentBuilder_Subject(t *testing.T) {
	builder := NewForgetPasswordContentBuilder("http://example.com/forget-password")

	testCases := []struct {
		name     string
		expected string
	}{
		{
			name:     "default subject",
			expected: "Восстановление пароля от аккаунта",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.Subject()
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestForgetPasswordContentBuilder_Body(t *testing.T) {
	builder := NewForgetPasswordContentBuilder("http://example.com/forget-password")

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
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/forget-password/MQ">ссылке</a>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
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
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/forget-password/MTIz">ссылке</a>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
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
<p>На данный email было запрошено письмо для восстановления забытого пароля.</p>
<p>Пожалуйста, перейдите по <a href="http://example.com/forget-password/OTg3NjU0MzIx">ссылке</a>, чтобы сменить пароль!</p>
<p>Если это были не Вы - проигнорируйте данное письмо!</p>
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
