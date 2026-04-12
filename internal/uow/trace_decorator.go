package uow

import (
	"context"

	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.UnitOfWork
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.UnitOfWork,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) Do(
	ctx context.Context,
	fn func(ctx context.Context, tx pg.Transaction) error,
) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.Do(ctx, fn)
}
