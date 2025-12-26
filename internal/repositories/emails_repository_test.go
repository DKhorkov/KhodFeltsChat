package repositories

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/contentbuilders"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/stretchr/testify/require"
)

func TestEmailsRepository_Send(t *testing.T) {
	t.Parallel()

	// Настройка SMTP конфигурации
	smtpConfig := config.SMTPConfig{
		Host: "smtp.freesmtpservers.com",
		Port: 25,
	}

	testCases := []struct {
		name          string
		subject       string
		body          string
		recipients    []string
		errorExpected bool
	}{
		{
			name:          "dialer error",
			subject:       "Test Subject",
			body:          "<h1>Test Body</h1>",
			recipients:    []string{"recipient1@example.com"},
			errorExpected: true,
		},
	}

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail: contentbuilders.NewVerifyEmailContentBuilder(
			"test",
		),
		ForgetPassword: contentbuilders.NewForgetPasswordContentBuilder(
			"test",
		),
	}

	repo := NewEmailsRepository(
		smtpConfig,
		contentBuilders,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := repo.send(context.Background(), tc.subject, tc.body, tc.recipients)
			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
