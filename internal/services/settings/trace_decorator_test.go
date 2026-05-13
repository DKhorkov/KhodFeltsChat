package settings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/settings"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNewTraceDecorator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.SettingsService
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
			base: mockservices.NewMockSettingsService(gomock.NewController(t)),
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
			base:          mockservices.NewMockSettingsService(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := settings.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_GetSettingsByUserID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name             string
		userID           uint64
		setupMocks       func(*mocktracing.MockProvider, *mockservices.MockSettingsService, *mocktracing.MockSpan)
		expectedSettings *domains.Settings
		expectedError    error
	}{
		{
			name:   "successful get settings with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockSettingsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(1)).
					Return(&domains.Settings{
						ID:        1,
						UserID:    1,
						Theme:     domains.ThemeLight,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedSettings: &domains.Settings{
				ID:        1,
				UserID:    1,
				Theme:     domains.ThemeLight,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:   "get settings error with tracing",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockSettingsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetSettingsByUserID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("not found"))
			},
			expectedSettings: nil,
			expectedError:    errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockSettingsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := settings.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.GetSettingsByUserID(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSettings, result)
		})
	}
}

func TestTraceDecorator_UpdateSettings(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name             string
		settings         domains.Settings
		setupMocks       func(*mocktracing.MockProvider, *mockservices.MockSettingsService, *mocktracing.MockSpan)
		expectedSettings *domains.Settings
		expectedError    error
	}{
		{
			name:     "successful update settings with tracing",
			settings: domains.Settings{UserID: 1, Theme: domains.ThemeDark},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockSettingsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					UpdateSettings(gomock.Any(), domains.Settings{UserID: 1, Theme: domains.ThemeDark}).
					Return(&domains.Settings{
						ID:        1,
						UserID:    1,
						Theme:     domains.ThemeDark,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedSettings: &domains.Settings{
				ID:        1,
				UserID:    1,
				Theme:     domains.ThemeDark,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:     "update settings error with tracing",
			settings: domains.Settings{UserID: 1, Theme: domains.ThemeDark},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockSettingsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateSettings(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedSettings: nil,
			expectedError:    errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockSettingsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := settings.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.UpdateSettings(context.Background(), tt.settings)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSettings, result)
		})
	}
}
