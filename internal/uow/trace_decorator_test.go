package uow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/uow"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNewTraceDecorator(t *testing.T) {
	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.UnitOfWork
	}{
		{
			name:          "create trace decorator with valid params",
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
			base: mockuow.NewMockUnitOfWork(gomock.NewController(t)),
		},
		{
			name:          "create trace decorator with nil base",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig:    tracing.SpanConfig{},
			base:          nil,
		},
		{
			name:          "create trace decorator with nil provider",
			traceProvider: nil,
			spanConfig:    tracing.SpanConfig{},
			base:          mockuow.NewMockUnitOfWork(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decorator := uow.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_Do(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mocktracing.MockProvider, *mockuow.MockUnitOfWork, *mocktracing.MockSpan)
		fn            func(ctx context.Context, tx pg.Transaction) error
		expectedError error
	}{
		{
			name: "successful execution with tracing",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				// Ожидаем создание span
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				// Ожидаем вызов базового Do
				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context, tx pg.Transaction) error) error {
						return fn(ctx, nil)
					})
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return nil
			},
			expectedError: nil,
		},
		{
			name: "base Do returns error",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return nil
			},
			expectedError: errors.New("database error"),
		},
		{
			name: "function returns error",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(ctx context.Context, tx pg.Transaction) error) error {
						return fn(ctx, nil)
					})
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return errors.New("business logic error")
			},
			expectedError: errors.New("business logic error"),
		},
		{
			name: "span creation with custom config",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
						assert.NotEmpty(t, name)

						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return nil
			},
			expectedError: nil,
		},
		{
			name: "with custom span events",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return nil
			},
			expectedError: nil,
		},
		{
			name: "trace provider returns error span",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockuow.MockUnitOfWork,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			fn: func(ctx context.Context, tx pg.Transaction) error {
				return nil
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockuow.NewMockUnitOfWork(ctrl)
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
			if tt.name == "with custom span events" {
				spanConfig.Events.Start.Name = "custom_start_event"
				spanConfig.Events.End.Name = "custom_end_event"
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := uow.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.Do(ctx, tt.fn)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
