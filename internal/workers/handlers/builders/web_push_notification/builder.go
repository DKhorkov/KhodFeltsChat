package web_push_notification

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
		var dto domains.WebPushNotificationDTO
		if err := json.Unmarshal(message.Data, &dto); err != nil {
			logging.LogError(b.logger, "Failed to unmarshal web push notification message", err)

			return
		}

		switch dto.Type {
		case domains.WebPushTypeNewMessage:
			b.handleNewMessage(ctx, dto)
		default:
			logging.LogError(
				b.logger,
				fmt.Sprintf("Unknown web push notification type: %s", dto.Type),
				nil,
			)
		}
	}
}

func (b *Builder) handleNewMessage(ctx context.Context, dto domains.WebPushNotificationDTO) {
	settings, err := b.settingsUseCases.GetSettingsByUserID(ctx, dto.UserID)
	if err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to get settings for User with ID=%d", dto.UserID),
			err,
		)

		return
	}

	if !domains.HasConsent(settings.WebPushConsents, domains.ConsentNewMessage) {
		return
	}

	var payload domains.NewMessagePayload
	if err = json.Unmarshal(dto.Payload, &payload); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal new message payload", err)

		return
	}

	if err = b.notificationsUseCases.SendNewMessageByWebPush(
		ctx,
		dto.UserID,
		payload,
	); err != nil {
		logging.LogError(
			b.logger,
			fmt.Sprintf("Failed to send web push to User with ID=%d", dto.UserID),
			err,
		)
	}
}
