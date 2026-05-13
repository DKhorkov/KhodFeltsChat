package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/libs/tracing"
)

type TraceDecorator struct {
	traceProvider tracing.Provider
	spanConfig    tracing.SpanConfig
	base          interfaces.SettingsUseCases
}

func NewTraceDecorator(
	traceProvider tracing.Provider,
	spanConfig tracing.SpanConfig,
	base interfaces.SettingsUseCases,
) *TraceDecorator {
	return &TraceDecorator{
		traceProvider: traceProvider,
		spanConfig:    spanConfig,
		base:          base,
	}
}

func (d *TraceDecorator) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.GetSettingsByUserID(ctx, userID)
}

func (d *TraceDecorator) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) (*domains.Settings, error) {
	ctx, span := d.traceProvider.Span(ctx, tracing.CallerName(tracing.DefaultSkipLevel))
	defer span.End()

	span.AddEvent(d.spanConfig.Events.Start.Name, d.spanConfig.Events.Start.Opts...)
	defer span.AddEvent(d.spanConfig.Events.End.Name, d.spanConfig.Events.End.Opts...)

	return d.base.UpdateSettings(ctx, settings)
}
