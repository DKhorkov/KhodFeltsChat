package interfaces

import (
	"context"

	"github.com/nats-io/nats.go"
)

type MessageHandler func(message *nats.Msg)

//go:generate mockgen -source=workers.go -destination=../../mocks/workers/message_handler_builder.go -package=mockworkers -exclude_interfaces=
type MessageHandlerBuilder interface {
	MessageHandler(ctx context.Context) MessageHandler
}
