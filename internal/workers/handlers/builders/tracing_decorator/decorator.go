package tracing_decorator

import (
	"context"

	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
	"github.com/nats-io/nats.go"
)

type Decorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.MessageHandlerBuilder
}

func New(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.MessageHandlerBuilder,
) *Decorator {
	return &Decorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *Decorator) MessageHandler(ctx context.Context) interfaces.MessageHandler {
	return func(message *nats.Msg) {
		ctx, span := d.traceProvider.Span( //nolint:govet // неважное затенение
			ctx,
			tracing.CallerName(tracing.DefaultSkipLevel),
		)
		defer span.End()

		span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
		defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

		d.base.MessageHandler(ctx)(message)
	}
}
