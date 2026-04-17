package notifications

import (
	"context"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	notificationsService interfaces.NotificationsService
	usersService         interfaces.UsersService
}

func New(
	notificationsService interfaces.NotificationsService,
	usersService interfaces.UsersService,
) *UseCases {
	return &UseCases{
		notificationsService: notificationsService,
		usersService:         usersService,
	}
}

func (u *UseCases) SendVerifyEmailMessage(ctx context.Context, userID uint64) error {
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.EmailConfirmed {
		return customerrors.ErrEmailAlreadyConfirmed
	}

	return u.notificationsService.SendVerifyEmailMessage(ctx, *user)
}

func (u *UseCases) SendForgetPasswordMessage(ctx context.Context, userID uint64) error {
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.EmailConfirmed {
		return customerrors.ErrEmailNotConfirmed
	}

	return u.notificationsService.SendForgetPasswordMessage(ctx, *user)
}
