package web_push_subscriptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/web_push_subscriptions"
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
		base          interfaces.WebPushSubscriptionsService
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
			base: mockservices.NewMockWebPushSubscriptionsService(gomock.NewController(t)),
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
			base:          mockservices.NewMockWebPushSubscriptionsService(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := web_push_subscriptions.NewTraceDecorator(
				tt.traceProvider,
				tt.spanConfig,
				tt.base,
			)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_CreateWebPushSubscription(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name                 string
		subscription         domains.WebPushSubscription
		setupMocks           func(*mocktracing.MockProvider, *mockservices.MockWebPushSubscriptionsService, *mocktracing.MockSpan)
		expectedSubscription *domains.WebPushSubscription
		expectedError        error
	}{
		{
			name: "successful create with tracing",
			subscription: domains.WebPushSubscription{
				UserID:        1,
				Endpoint:      "https://example.com/push",
				EncryptionKey: "key1",
				Auth:          "auth1",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					CreateWebPushSubscription(gomock.Any(), domains.WebPushSubscription{
						UserID:        1,
						Endpoint:      "https://example.com/push",
						EncryptionKey: "key1",
						Auth:          "auth1",
					}).
					Return(&domains.WebPushSubscription{
						ID:            1,
						UserID:        1,
						Endpoint:      "https://example.com/push",
						EncryptionKey: "key1",
						Auth:          "auth1",
						CreatedAt:     now,
					}, nil)
			},
			expectedSubscription: &domains.WebPushSubscription{
				ID:            1,
				UserID:        1,
				Endpoint:      "https://example.com/push",
				EncryptionKey: "key1",
				Auth:          "auth1",
				CreatedAt:     now,
			},
			expectedError: nil,
		},
		{
			name: "create error with tracing",
			subscription: domains.WebPushSubscription{
				UserID:        1,
				Endpoint:      "https://example.com/push",
				EncryptionKey: "key1",
				Auth:          "auth1",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateWebPushSubscription(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedSubscription: nil,
			expectedError:        errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockWebPushSubscriptionsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := web_push_subscriptions.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.CreateWebPushSubscription(
				context.Background(),
				tt.subscription,
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSubscription, result)
		})
	}
}

func TestTraceDecorator_GetWebPushSubscriptionsByUserID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name                  string
		userID                uint64
		setupMocks            func(*mocktracing.MockProvider, *mockservices.MockWebPushSubscriptionsService, *mocktracing.MockSpan)
		expectedSubscriptions []domains.WebPushSubscription
		expectedError         error
	}{
		{
			name:   "successful get with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(1)).
					Return([]domains.WebPushSubscription{
						{
							ID:            1,
							UserID:        1,
							Endpoint:      "https://example.com/push",
							EncryptionKey: "key1",
							Auth:          "auth1",
							CreatedAt:     now,
						},
					}, nil)
			},
			expectedSubscriptions: []domains.WebPushSubscription{
				{
					ID:            1,
					UserID:        1,
					Endpoint:      "https://example.com/push",
					EncryptionKey: "key1",
					Auth:          "auth1",
					CreatedAt:     now,
				},
			},
			expectedError: nil,
		},
		{
			name:   "get error with tracing",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetWebPushSubscriptionsByUserID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("not found"))
			},
			expectedSubscriptions: nil,
			expectedError:         errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockWebPushSubscriptionsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := web_push_subscriptions.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.GetWebPushSubscriptionsByUserID(
				context.Background(),
				tt.userID,
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedSubscriptions, result)
		})
	}
}

func TestTraceDecorator_DeleteWebPushSubscription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            uint64
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockWebPushSubscriptionsService, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful delete with tracing",
			id:   1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "delete error with tracing",
			id:   1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockWebPushSubscriptionsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					DeleteWebPushSubscription(gomock.Any(), uint64(1)).
					Return(errors.New("database error"))
			},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockWebPushSubscriptionsService(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := web_push_subscriptions.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			err := decorator.DeleteWebPushSubscription(context.Background(), tt.id)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
