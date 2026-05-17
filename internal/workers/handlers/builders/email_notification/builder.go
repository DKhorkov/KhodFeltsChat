package email_notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/logging"
	"github.com/nats-io/nats.go"
)

type Builder struct {
	notificationsUseCases interfaces.NotificationsUseCases
	settingsUseCases      interfaces.SettingsUseCases
	logger                logging.Logger
}

func New(
	notificationsUseCases interfaces.NotificationsUseCases,
	settingsUseCases interfaces.SettingsUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		notificationsUseCases: notificationsUseCases,
		settingsUseCases:      settingsUseCases,
		logger:                logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		var dto domains.EmailNotificationDTO
		if err := json.Unmarshal(message.Data, &dto); err != nil {
			logging.LogError(b.logger, "Failed to unmarshal email notification message", err)

			return
		}

		switch dto.Type {
		case domains.EmailTypeVerifyEmail:
			b.handleVerifyEmail(ctx, dto)
		case domains.EmailTypeForgetPassword:
			b.handleForgetPassword(ctx, dto)
		case domains.EmailTypeNewMessage:
			b.handleNewMessage(ctx, dto)
		default:
			logging.LogError(
				b.logger,
				fmt.Sprintf("Unknown email notification type: %s", dto.Type),
				nil,
			)
		}
	}
}

func (b *Builder) handleVerifyEmail(ctx context.Context, dto domains.EmailNotificationDTO) {
	if err := b.notificationsUseCases.SendVerifyEmailMessage(ctx, dto.UserID); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send verify-email to User with ID=%d", dto.UserID),
			err,
		)
	}
}

func (b *Builder) handleForgetPassword(ctx context.Context, dto domains.EmailNotificationDTO) {
	if err := b.notificationsUseCases.SendForgetPasswordMessage(ctx, dto.UserID); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send forget-password to User with ID=%d", dto.UserID),
			err,
		)
	}
}

func (b *Builder) handleNewMessage(ctx context.Context, dto domains.EmailNotificationDTO) {
	settings, err := b.settingsUseCases.GetSettingsByUserID(ctx, dto.UserID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get settings for User with ID=%d", dto.UserID),
			err,
		)

		return
	}

	if !domains.HasConsent(settings.EmailConsents, domains.ConsentNewMessage) {
		return
	}

	var payload domains.NewMessagePayload
	if err = json.Unmarshal(dto.Payload, &payload); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal new message payload", err)

		return
	}

	if err = b.notificationsUseCases.SendNewMessageByEmail(ctx, dto.UserID, payload); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send new-message email to User with ID=%d", dto.UserID),
			err,
		)
	}
}
