package notifications

import (
	"context"
	"errors"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
)

type UseCases struct {
	notificationsService        interfaces.NotificationsService
	usersService                interfaces.UsersService
	messagesService             interfaces.MessagesService
	chatsService                interfaces.ChatsService
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService
	logger                      logging.Logger
}

func New(
	notificationsService interfaces.NotificationsService,
	usersService interfaces.UsersService,
	messagesService interfaces.MessagesService,
	chatsService interfaces.ChatsService,
	webPushSubscriptionsService interfaces.WebPushSubscriptionsService,
	logger logging.Logger,
) *UseCases {
	return &UseCases{
		notificationsService:        notificationsService,
		usersService:                usersService,
		messagesService:             messagesService,
		chatsService:                chatsService,
		webPushSubscriptionsService: webPushSubscriptionsService,
		logger:                      logger,
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

func (u *UseCases) SendNewMessageByEmail(
	ctx context.Context,
	userID uint64,
	payload domains.NewMessagePayload,
) error {
	user, err := u.usersService.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	message, err := u.messagesService.GetMessageByID(ctx, userID, payload.MessageID)
	if err != nil {
		return err
	}

	chat, err := u.chatsService.GetChatByID(ctx, message.ChatID, userID)
	if err != nil {
		return err
	}

	return u.notificationsService.SendNewMessageByEmail(ctx, *user, *message, *chat)
}

func (u *UseCases) SendNewMessageByWebPush(
	ctx context.Context,
	userID uint64,
	payload domains.NewMessagePayload,
) error {
	message, err := u.messagesService.GetMessageByID(ctx, userID, payload.MessageID)
	if err != nil {
		return err
	}

	subscriptions, err := u.webPushSubscriptionsService.GetWebPushSubscriptionsByUserID(
		ctx,
		userID,
	)
	if err != nil {
		return err
	}

	unreadCount, err := u.messagesService.GetUserUnreadCount(ctx, userID)
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		if err = u.notificationsService.SendNewMessageByWebPush(
			ctx,
			sub,
			*message,
			unreadCount,
		); err != nil {
			if errors.Is(err, customerrors.ErrWebPushSubscriptionExpired) {
				u.logger.Info(
					fmt.Sprintf(
						"Web push subscription ID=%d for User ID=%d expired, deleting",
						sub.ID,
						userID,
					),
				)

				if err = u.webPushSubscriptionsService.DeleteWebPushSubscription(
					ctx,
					sub.ID,
				); err != nil {
					logging.LogError(
						u.logger,
						fmt.Sprintf(
							"Failed to delete expired web push subscription ID=%d",
							sub.ID,
						),
						err,
					)
				}

				continue
			}

			logging.LogError(
				u.logger,
				fmt.Sprintf(
					"Failed to send web push notification to endpoint=%s for User with ID=%d",
					sub.Endpoint,
					userID,
				),
				err,
			)
		}
	}

	return nil
}
