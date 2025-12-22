package repositories

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/stretchr/testify/require"
)

func TestEmailsRepository_Send(t *testing.T) {
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

	repo := NewEmailsRepository(
		smtpConfig,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := repo.send(context.Background(), tc.subject, tc.body, tc.recipients)
			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
