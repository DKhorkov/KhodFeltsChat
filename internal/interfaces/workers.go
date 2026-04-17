package interfaces

import (
	"context"

	"github.com/nats-io/nats.go"
)

type MessageHandler func(message *nats.Msg)

//go:generate mockgen -source=usecases.go -destination=../../mocks/usecases/users_usecases.go -package=mockusecases -exclude_interfaces=AuthUseCases,ChatsUseCases,MessagesUseCases,NotificationsUseCases
type MessageHandlerBuilder interface {
	MessageHandler(ctx context.Context) MessageHandler
}
