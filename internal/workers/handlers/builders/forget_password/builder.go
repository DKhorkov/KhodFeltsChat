package forget_password

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
		forgetPasswordDTO := b.natsMessageToDTO(message)
		if forgetPasswordDTO == nil {
			return
		}

		if err := b.useCases.SendForgetPasswordMessage(
			ctx,
			forgetPasswordDTO.UserID,
		); err != nil {
			logging.LogError(
				b.logger,
				fmt.Sprintf(
					"Failed to send forget-password message to User with ID=%d ",
					forgetPasswordDTO.UserID,
				),
				err,
			)
		}
	}
}

func (b *Builder) natsMessageToDTO(
	message *nats.Msg,
) *domains.ForgetPasswordNotificationDTO {
	var forgetPasswordDTO domains.ForgetPasswordNotificationDTO
	if err := json.Unmarshal(message.Data, &forgetPasswordDTO); err != nil {
		logging.LogError(b.logger, "Failed to unmarshal forget-password message", err)

		return nil
	}

	return &forgetPasswordDTO
}
