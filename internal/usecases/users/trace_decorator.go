package users

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.UsersUseCases
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.UsersUseCases,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetUsers(ctx, filters, pagination)
}

func (d *TraceDecorator) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetUserByID(ctx, id)
}

func (d *TraceDecorator) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (*domains.User, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateUser(ctx, userData)
}

func (d *TraceDecorator) UpdateAvatar(ctx context.Context, userID uint64, data io.Reader) (string, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateAvatar(ctx, userID, data)
}

func (d *TraceDecorator) DeleteAvatar(ctx context.Context, userID uint64) error {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.DeleteAvatar(ctx, userID)
}
