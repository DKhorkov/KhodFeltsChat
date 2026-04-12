package messages_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/usecases/messages"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/pointers"
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
		base          interfaces.MessagesUseCases
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
			base: mockusecases.NewMockMessagesUseCases(gomock.NewController(t)),
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
			base:          mockusecases.NewMockMessagesUseCases(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := messages.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_SaveMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name           string
		message        domains.Message
		setupMocks     func(*mocktracing.MockProvider, *mockusecases.MockMessagesUseCases, *mocktracing.MockSpan)
		expectedResult *domains.Message
		expectedError  error
	}{
		{
			name: "successful save message with tracing",
			message: domains.Message{
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "Hello, world!",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), domains.Message{
						ChatID:    1,
						Sender:    domains.User{ID: 1, Username: "user1"},
						Text:      "Hello, world!",
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    false,
					}).
					Return(&domains.Message{
						ID:        1,
						ChatID:    1,
						Sender:    domains.User{ID: 1, Username: "user1"},
						Text:      "Hello, world!",
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    false,
					}, nil)
			},
			expectedResult: &domains.Message{
				ID:        1,
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "Hello, world!",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			expectedError: nil,
		},
		{
			name: "save message with empty text",
			message: domains.Message{
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(&domains.Message{
						ID:        2,
						ChatID:    1,
						Sender:    domains.User{ID: 1, Username: "user1"},
						Text:      "",
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    false,
					}, nil)
			},
			expectedResult: &domains.Message{
				ID:        2,
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			expectedError: nil,
		},
		{
			name: "save message with special characters",
			message: domains.Message{
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "Hello! @#$%^&*()_+",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(&domains.Message{
						ID:        3,
						ChatID:    1,
						Sender:    domains.User{ID: 1, Username: "user1"},
						Text:      "Hello! @#$%^&*()_+",
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    false,
					}, nil)
			},
			expectedResult: &domains.Message{
				ID:        3,
				ChatID:    1,
				Sender:    domains.User{ID: 1, Username: "user1"},
				Text:      "Hello! @#$%^&*()_+",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    false,
			},
			expectedError: nil,
		},
		{
			name: "base returns error",
			message: domains.Message{
				ChatID: 1,
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("failed to save message"))
			},
			expectedResult: nil,
			expectedError:  errors.New("failed to save message"),
		},
		{
			name: "chat not found",
			message: domains.Message{
				ChatID: 999,
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("chat not found"))
			},
			expectedResult: nil,
			expectedError:  errors.New("chat not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockMessagesUseCases(ctrl)
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

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			result, err := decorator.SaveMessage(ctx, tt.message)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestTraceDecorator_GetChatMessages(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name             string
		userID           uint64
		chatID           uint64
		pagination       *domains.Pagination
		setupMocks       func(*mocktracing.MockProvider, *mockusecases.MockMessagesUseCases, *mocktracing.MockSpan)
		expectedMessages []domains.Message
		expectedError    error
	}{
		{
			name:   "successful get chat messages with tracing",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](10),
						Offset: pointers.New[uint64](0),
					}).
					Return([]domains.Message{
						{
							ID:        1,
							ChatID:    1,
							Sender:    domains.User{ID: 2, Username: "user2"},
							Text:      "Hello!",
							CreatedAt: now,
							UpdatedAt: now,
							IsRead:    true,
						},
						{
							ID:        2,
							ChatID:    1,
							Sender:    domains.User{ID: 1, Username: "user1"},
							Text:      "Hi there!",
							CreatedAt: now,
							UpdatedAt: now,
							IsRead:    true,
						},
					}, nil)
			},
			expectedMessages: []domains.Message{
				{
					ID:        1,
					ChatID:    1,
					Sender:    domains.User{ID: 2, Username: "user2"},
					Text:      "Hello!",
					CreatedAt: now,
					UpdatedAt: now,
					IsRead:    true,
				},
				{
					ID:        2,
					ChatID:    1,
					Sender:    domains.User{ID: 1, Username: "user1"},
					Text:      "Hi there!",
					CreatedAt: now,
					UpdatedAt: now,
					IsRead:    true,
				},
			},
			expectedError: nil,
		},
		{
			name:   "get chat messages with default pagination",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](20),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](20),
						Offset: pointers.New[uint64](0),
					}).
					Return([]domains.Message{}, nil)
			},
			expectedMessages: []domains.Message{},
			expectedError:    nil,
		},
		{
			name:   "get chat messages with offset",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](50),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](10),
						Offset: pointers.New[uint64](50),
					}).
					Return([]domains.Message{
						{ID: 51, Text: "Message 51"},
						{ID: 52, Text: "Message 52"},
					}, nil)
			},
			expectedMessages: []domains.Message{
				{ID: 51, Text: "Message 51"},
				{ID: 52, Text: "Message 52"},
			},
			expectedError: nil,
		},
		{
			name:   "empty messages list",
			userID: 1,
			chatID: 2,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(2), gomock.Any()).
					Return([]domains.Message{}, nil)
			},
			expectedMessages: []domains.Message{},
			expectedError:    nil,
		},
		{
			name:   "chat not found",
			userID: 1,
			chatID: 999,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(999), gomock.Any()).
					Return(nil, errors.New("chat not found"))
			},
			expectedMessages: nil,
			expectedError:    errors.New("chat not found"),
		},
		{
			name:   "user not member of chat",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), gomock.Any()).
					Return(nil, errors.New("user is not a member of this chat"))
			},
			expectedMessages: nil,
			expectedError:    errors.New("user is not a member of this chat"),
		},
		{
			name:   "database error",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			expectedMessages: nil,
			expectedError:    errors.New("database connection failed"),
		},
		{
			name:       "nil pagination",
			userID:     1,
			chatID:     1,
			pagination: nil,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockMessagesUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), nil).
					Return([]domains.Message{}, nil)
			},
			expectedMessages: []domains.Message{},
			expectedError:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockMessagesUseCases(ctrl)
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

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			result, err := decorator.GetChatMessages(ctx, tt.userID, tt.chatID, tt.pagination)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedMessages, result)
		})
	}
}
