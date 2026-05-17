package web_push_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	webpush "github.com/DKhorkov/kfc/internal/repositories/web_push"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
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
		base          interfaces.WebPushRepository
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
			base: mockrepositories.NewMockWebPushRepository(gomock.NewController(t)),
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
			base:          mockrepositories.NewMockWebPushRepository(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := webpush.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_SendNotification(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		subscription  domains.WebPushSubscription
		message       domains.Message
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockWebPushRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful send notification with tracing",
			subscription: domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://fcm.googleapis.com/fcm/send/test",
				EncryptionKey: "test-p256dh",
				Auth:          "test-auth",
				CreatedAt:     now,
			},
			message: domains.Message{
				ID:     10,
				ChatID: 5,
				Text:   "Hello!",
				Sender: domains.User{
					ID:       2,
					Username: "alice",
				},
				CreatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockWebPushRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SendNotification(gomock.Any(), domains.WebPushSubscription{
						ID:            1,
						UserID:        1,
						Endpoint:      "https://fcm.googleapis.com/fcm/send/test",
						EncryptionKey: "test-p256dh",
						Auth:          "test-auth",
						CreatedAt:     now,
					}, domains.Message{
						ID:     10,
						ChatID: 5,
						Text:   "Hello!",
						Sender: domains.User{
							ID:       2,
							Username: "alice",
						},
						CreatedAt: now,
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "send notification error with tracing",
			subscription: domains.WebPushSubscription{
				ID:            2,
				UserID:        1,
				Endpoint:      "https://fcm.googleapis.com/fcm/send/expired",
				EncryptionKey: "test-p256dh",
				Auth:          "test-auth",
			},
			message: domains.Message{
				ID:     11,
				ChatID: 5,
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockWebPushRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("web-push subscription expired"))
			},
			expectedError: errors.New("web-push subscription expired"),
		},
		{
			name: "send notification network error with tracing",
			subscription: domains.WebPushSubscription{
				ID:            3,
				UserID:        2,
				Endpoint:      "https://push.example.com/send/test",
				EncryptionKey: "key",
				Auth:          "auth",
			},
			message: domains.Message{
				ID:     12,
				ChatID: 3,
				Text:   "Network test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockWebPushRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SendNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("connection refused"))
			},
			expectedError: errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockWebPushRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := webpush.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			err := decorator.SendNotification(context.Background(), tt.subscription, tt.message)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
