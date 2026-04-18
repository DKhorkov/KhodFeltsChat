package tracing_decorator_test

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/workers/handlers/builders/tracing_decorator"
	mockworkers "github.com/DKhorkov/kfc/mocks/workers"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.MessageHandlerBuilder
	}{
		{
			name:          "create decorator with valid params",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig: tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{
						Name: "start",
						Opts: []trace.EventOption{},
					},
					End: tracing.SpanEventConfig{
						Name: "end",
						Opts: []trace.EventOption{},
					},
				},
			},
			base: mockworkers.NewMockMessageHandlerBuilder(gomock.NewController(t)),
		},
		{
			name:          "create decorator with nil base",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig:    tracing.SpanConfig{},
			base:          nil,
		},
		{
			name:          "create decorator with nil provider",
			traceProvider: nil,
			spanConfig:    tracing.SpanConfig{},
			base:          mockworkers.NewMockMessageHandlerBuilder(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := tracing_decorator.New(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestDecorator_MessageHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mocktracing.MockProvider, *mockworkers.MockMessageHandlerBuilder, *mocktracing.MockSpan)
		message    *nats.Msg
	}{
		{
			name: "successful message handler with tracing",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockworkers.MockMessageHandlerBuilder,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					MessageHandler(gomock.Any()).
					DoAndReturn(func(_ context.Context) interfaces.MessageHandler {
						return func(_ *nats.Msg) {
							// base handler logic
						}
					})
			},
			message: &nats.Msg{
				Subject: "test.subject",
				Data:    []byte("test data"),
			},
		},
		{
			name: "message handler with custom span events",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockworkers.MockMessageHandlerBuilder,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					MessageHandler(gomock.Any()).
					Return(func(_ *nats.Msg) {})
			},
			message: &nats.Msg{
				Subject: "another.subject",
				Data:    []byte("another data"),
			},
		},
		{
			name: "message with empty data",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockworkers.MockMessageHandlerBuilder,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					MessageHandler(gomock.Any()).
					Return(func(_ *nats.Msg) {})
			},
			message: &nats.Msg{
				Subject: "empty.data",
				Data:    []byte{},
			},
		},
		{
			name: "message with nil data",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockworkers.MockMessageHandlerBuilder,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					MessageHandler(gomock.Any()).
					Return(func(_ *nats.Msg) {})
			},
			message: &nats.Msg{
				Subject: "nil.data",
				Data:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockworkers.NewMockMessageHandlerBuilder(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{
						Name: "start",
						Opts: []trace.EventOption{},
					},
					End: tracing.SpanEventConfig{
						Name: "end",
						Opts: []trace.EventOption{},
					},
				},
			}

			// Для теста с кастомными событиями
			if tt.name == "message handler with custom span events" {
				spanConfig.Events.Start.Name = "custom_start_event"
				spanConfig.Events.End.Name = "custom_end_event"
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := tracing_decorator.New(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			handler := decorator.MessageHandler(ctx)

			// Вызываем хендлер
			assert.NotPanics(t, func() {
				handler(tt.message)
			})
		})
	}
}
