package push_notification

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
	pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases
	messagesUseCases          interfaces.MessagesUseCases
	logger                    logging.Logger
}

func New(
	pushSubscriptionsUseCases interfaces.PushSubscriptionsUseCases,
	messagesUseCases interfaces.MessagesUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		pushSubscriptionsUseCases: pushSubscriptionsUseCases,
		messagesUseCases:          messagesUseCases,
		logger:                    logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		dto := b.natsMessageToDTO(message)
		if dto == nil {
			return
		}

		msg, err := b.messagesUseCases.GetMessageByID(ctx, dto.UserID, dto.MessageID)
		if err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf(
					"Failed to get message with ID=%d",
					dto.MessageID,
				),
				err,
			)

			return
		}

		subscriptions, err := b.pushSubscriptionsUseCases.GetPushSubscriptionsByUserID(ctx, dto.UserID)
		if err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf(
					"Failed to get push subscriptions for User with ID=%d",
					dto.UserID,
				),
				err,
			)

			return
		}

		for _, sub := range subscriptions {
			if err = b.pushSubscriptionsUseCases.SendPushNotification(ctx, sub, *msg); err != nil {
				logging.LogError(
					b.logger,
					fmt.Sprintf(
						"Failed to send push notification to endpoint=%s for User with ID=%d",
						sub.Endpoint,
						dto.UserID,
					),
					err,
				)
			}
		}
	}
}

func (b *Builder) natsMessageToDTO(message *nats.Msg) *domains.PushNotificationDTO {
	var dto domains.PushNotificationDTO
	if err := json.Unmarshal(message.Data, &dto); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal push-notification message", err)

		return nil
	}

	return &dto
}
