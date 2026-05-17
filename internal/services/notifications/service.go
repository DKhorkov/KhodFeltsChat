package notifications

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type Service struct {
	emailsRepository  interfaces.EmailsRepository
	webPushRepository interfaces.WebPushRepository
}

func New(
	emailsRepository interfaces.EmailsRepository,
	webPushRepository interfaces.WebPushRepository,
) *Service {
	return &Service{
		emailsRepository:  emailsRepository,
		webPushRepository: webPushRepository,
	}
}

func (s *Service) SendVerifyEmailMessage(ctx context.Context, user domains.User) error {
	return s.emailsRepository.SendVerifyEmailMessage(ctx, user)
}

func (s *Service) SendForgetPasswordMessage(ctx context.Context, user domains.User) error {
	return s.emailsRepository.SendForgetPasswordMessage(ctx, user)
}

func (s *Service) SendNewMessageByEmail(
	ctx context.Context,
	recipient domains.User,
	message domains.Message,
	chat domains.Chat,
) error {
	return s.emailsRepository.SendMessage(ctx, recipient, message, chat)
}

func (s *Service) SendNewMessageByWebPush(
	ctx context.Context,
	subscription domains.WebPushSubscription,
	message domains.Message,
) error {
	return s.webPushRepository.SendNotification(ctx, subscription, message)
}
