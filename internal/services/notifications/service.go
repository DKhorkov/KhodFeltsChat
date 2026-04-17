package notifications

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type Service struct {
	emailsRepository interfaces.EmailsRepository
}

func New(
	emailsRepository interfaces.EmailsRepository,
) *Service {
	return &Service{
		emailsRepository: emailsRepository,
	}
}

func (s *Service) SendVerifyEmailMessage(ctx context.Context, user domains.User) error {
	return s.emailsRepository.SendVerifyEmailMessage(ctx, user)
}

func (s *Service) SendForgetPasswordMessage(ctx context.Context, user domains.User) error {
	return s.emailsRepository.SendForgetPasswordMessage(ctx, user)
}
