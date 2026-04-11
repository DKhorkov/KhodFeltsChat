package emails

import (
	"context"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"gopkg.in/gomail.v2"
)

type Repository struct {
	smtpConfig      config.SMTPConfig
	contentBuilders interfaces.ContentBuilders
}

func New(
	smtpConfig config.SMTPConfig,
	contentBuilders interfaces.ContentBuilders,
) *Repository {
	return &Repository{
		smtpConfig:      smtpConfig,
		contentBuilders: contentBuilders,
	}
}

func (repo *Repository) SendVerifyEmailMessage(ctx context.Context, user domains.User) error {
	return repo.send(
		ctx,
		repo.contentBuilders.VerifyEmail.Subject(),
		repo.contentBuilders.VerifyEmail.Body(user),
		[]string{user.Email},
	)
}

func (repo *Repository) SendForgetPasswordMessage(
	ctx context.Context,
	user domains.User,
) error {
	return repo.send(
		ctx,
		repo.contentBuilders.ForgetPassword.Subject(),
		repo.contentBuilders.ForgetPassword.Body(user),
		[]string{user.Email},
	)
}

func (repo *Repository) send(
	_ context.Context,
	subject, body string,
	recipients []string,
) error {
	message := gomail.NewMessage()
	message.SetHeader("From", repo.smtpConfig.Login)
	message.SetHeader("To", recipients...)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body)

	smtpClient := gomail.NewDialer(
		repo.smtpConfig.Host,
		repo.smtpConfig.Port,
		repo.smtpConfig.Login,
		repo.smtpConfig.Password,
	)

	return smtpClient.DialAndSend(message)
}
