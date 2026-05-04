//go:build integration

package emails

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/contentbuilders/forget_password"
	"github.com/DKhorkov/kfc/internal/contentbuilders/verify_email"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	mockcache "github.com/DKhorkov/libs/cache/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRepository_SendVerifyEmailMessage(t *testing.T) {
	t.Parallel()

	// Настройка SMTP конфигурации
	smtpConfig := config.SMTPConfig{
		Host: "smtp.freesmtpservers.com",
		Port: 25,
	}

	testCases := []struct {
		name          string
		user          domains.User
		errorExpected bool
	}{
		{
			name:          "dialer error",
			user:          domains.User{Email: "alexqwerty35@yandex.ru"},
			errorExpected: true,
		},
	}

	ctrl := gomock.NewController(t)
	mc := mockcache.NewMockProvider(ctrl)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail:    verify_email.New("test", mc),
		ForgetPassword: forget_password.New(mc),
	}

	repo := New(
		smtpConfig,
		contentBuilders,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := repo.SendVerifyEmailMessage(context.Background(), tc.user)
			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRepository_SendForgetPasswordMessage(t *testing.T) {
	t.Parallel()

	// Настройка SMTP конфигурации
	smtpConfig := config.SMTPConfig{
		Host: "smtp.freesmtpservers.com",
		Port: 25,
	}

	testCases := []struct {
		name          string
		user          domains.User
		errorExpected bool
	}{
		{
			name:          "dialer error",
			user:          domains.User{Email: "alexqwerty35@yandex.ru"},
			errorExpected: true,
		},
	}

	ctrl := gomock.NewController(t)
	mc := mockcache.NewMockProvider(ctrl)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail:    verify_email.New("test", mc),
		ForgetPassword: forget_password.New(mc),
	}

	repo := New(
		smtpConfig,
		contentBuilders,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := repo.SendForgetPasswordMessage(context.Background(), tc.user)
			if tc.errorExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRepository_Send(t *testing.T) {
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

	ctrl := gomock.NewController(t)
	mc := mockcache.NewMockProvider(ctrl)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	contentBuilders := interfaces.ContentBuilders{
		VerifyEmail:    verify_email.New("test", mc),
		ForgetPassword: forget_password.New(mc),
	}

	repo := New(
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
