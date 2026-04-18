package verify_email

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
	useCases interfaces.NotificationsUseCases
	logger   logging.Logger
}

func New(
	useCases interfaces.NotificationsUseCases,
	logger logging.Logger,
) *Builder {
	return &Builder{
		useCases: useCases,
		logger:   logger,
	}
}

func (b *Builder) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		verifyEmailDTO := b.natsMessageToDTO(message)
		if verifyEmailDTO == nil {
			return
		}

		if err := b.useCases.SendVerifyEmailMessage(
			ctx,
			verifyEmailDTO.UserID,
		); err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf(
					"Failed to send verify-email message to User with ID=%d ",
					verifyEmailDTO.UserID,
				),
				err,
			)
		}
	}
}

func (b *Builder) natsMessageToDTO(message *nats.Msg) *domains.VerifyEmailNotificationDTO {
	var verifyEmailDTO domains.VerifyEmailNotificationDTO
	if err := json.Unmarshal(message.Data, &verifyEmailDTO); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal verify-email message", err)

		return nil
	}

	return &verifyEmailDTO
}
