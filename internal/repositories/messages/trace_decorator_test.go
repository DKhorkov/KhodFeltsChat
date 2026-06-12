package messages_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories/messages"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
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
		base          interfaces.MessagesRepository
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
			base: mockrepositories.NewMockMessagesRepository(gomock.NewController(t)),
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
			base:          mockrepositories.NewMockMessagesRepository(gomock.NewController(t)),
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
		name          string
		message       domains.Message
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedID    uint64
		expectedError error
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
				mockBase *mockrepositories.MockMessagesRepository,
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
					Return(uint64(1), nil)
			},
			expectedID:    1,
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
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(uint64(2), nil)
			},
			expectedID:    2,
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
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(uint64(3), nil)
			},
			expectedID:    3,
			expectedError: nil,
		},
		{
			name: "chat not found",
			message: domains.Message{
				ChatID: 999,
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("chat not found"))
			},
			expectedID:    0,
			expectedError: errors.New("chat not found"),
		},
		{
			name: "user not member of chat",
			message: domains.Message{
				ChatID: 1,
				Sender: domains.User{ID: 999, Username: "outsider"},
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("user is not a member of this chat"))
			},
			expectedID:    0,
			expectedError: errors.New("user is not a member of this chat"),
		},
		{
			name: "database error",
			message: domains.Message{
				ChatID: 1,
				Text:   "Test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					SaveMessage(gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("database connection failed"))
			},
			expectedID:    0,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			id, err := decorator.SaveMessage(ctx, tt.message)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedID, id)
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
		setupMocks       func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
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
				mockBase *mockrepositories.MockMessagesRepository,
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
			name:   "get chat messages with custom pagination",
			userID: 1,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](20),
				Offset: pointers.New[uint64](50),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(1), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](20),
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
				mockBase *mockrepositories.MockMessagesRepository,
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
				mockBase *mockrepositories.MockMessagesRepository,
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
			userID: 999,
			chatID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMessages(gomock.Any(), uint64(999), uint64(1), gomock.Any()).
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
				mockBase *mockrepositories.MockMessagesRepository,
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
				mockBase *mockrepositories.MockMessagesRepository,
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
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			messages, err := decorator.GetChatMessages(ctx, tt.userID, tt.chatID, tt.pagination)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedMessages, messages)
		})
	}
}

func TestTraceDecorator_GetMessageByID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name            string
		userID          uint64
		messageID       uint64
		setupMocks      func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedMessage *domains.Message
		expectedError   error
	}{
		{
			name:      "successful get message by id with tracing",
			userID:    1,
			messageID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(1)).
					Return(&domains.Message{
						ID:        1,
						ChatID:    1,
						Sender:    domains.User{ID: 2, Username: "user2"},
						Text:      "Hello!",
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    true,
					}, nil)
			},
			expectedMessage: &domains.Message{
				ID:        1,
				ChatID:    1,
				Sender:    domains.User{ID: 2, Username: "user2"},
				Text:      "Hello!",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    true,
			},
			expectedError: nil,
		},
		{
			name:      "message not found",
			userID:    1,
			messageID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
					Return(nil, errors.New("message not found"))
			},
			expectedMessage: nil,
			expectedError:   errors.New("message not found"),
		},
		{
			name:      "user not authorized to view message",
			userID:    1,
			messageID: 2,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(2)).
					Return(nil, errors.New("user not authorized to view this message"))
			},
			expectedMessage: nil,
			expectedError:   errors.New("user not authorized to view this message"),
		},
		{
			name:      "database error",
			userID:    1,
			messageID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetMessageByID(gomock.Any(), uint64(1), uint64(1)).
					Return(nil, errors.New("database connection failed"))
			},
			expectedMessage: nil,
			expectedError:   errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			message, err := decorator.GetMessageByID(ctx, tt.userID, tt.messageID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedMessage, message)
		})
	}
}

func TestTraceDecorator_ChangeMessagesIsReadStatus(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		messages      []domains.Message
		isRead        bool
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful change messages is read status to true with tracing",
			userID: 1,
			messages: []domains.Message{
				{ID: 1, ChatID: 1, IsRead: false, CreatedAt: now, UpdatedAt: now},
				{ID: 2, ChatID: 1, IsRead: false, CreatedAt: now, UpdatedAt: now},
			},
			isRead: true,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), []domains.Message{
						{ID: 1, ChatID: 1, IsRead: false, CreatedAt: now, UpdatedAt: now},
						{ID: 2, ChatID: 1, IsRead: false, CreatedAt: now, UpdatedAt: now},
					}, true).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "successful change messages is read status to false",
			userID: 1,
			messages: []domains.Message{
				{ID: 1, ChatID: 1, IsRead: true, CreatedAt: now, UpdatedAt: now},
			},
			isRead: false,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), []domains.Message{
						{ID: 1, ChatID: 1, IsRead: true, CreatedAt: now, UpdatedAt: now},
					}, false).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "empty messages list",
			userID:   1,
			messages: []domains.Message{},
			isRead:   true,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), []domains.Message{}, true).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "message not found",
			userID: 1,
			messages: []domains.Message{
				{ID: 999, ChatID: 1},
			},
			isRead: true,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), gomock.Any(), true).
					Return(errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name:   "user not authorized",
			userID: 999,
			messages: []domains.Message{
				{ID: 1, ChatID: 1},
			},
			isRead: true,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(999), gomock.Any(), true).
					Return(errors.New("user not authorized to change message status"))
			},
			expectedError: errors.New("user not authorized to change message status"),
		},
		{
			name:   "database error",
			userID: 1,
			messages: []domains.Message{
				{ID: 1, ChatID: 1},
			},
			isRead: true,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), gomock.Any(), true).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ChangeMessagesIsReadStatus(ctx, tt.userID, tt.messages, tt.isRead)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_DeleteMessageForUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		messageID     uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:      "successful delete message for user with tracing",
			userID:    1,
			messageID: 10,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					DeleteMessageForUser(gomock.Any(), uint64(1), uint64(10)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "message not found",
			userID:    1,
			messageID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					DeleteMessageForUser(gomock.Any(), uint64(1), uint64(999)).
					Return(errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name:      "database error",
			userID:    1,
			messageID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					DeleteMessageForUser(gomock.Any(), uint64(1), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.DeleteMessageForUser(ctx, tt.userID, tt.messageID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_DeleteMessageForAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		messageID     uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:      "successful delete message for all with tracing",
			messageID: 10,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					DeleteMessageForAll(gomock.Any(), uint64(10)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "message not found",
			messageID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					DeleteMessageForAll(gomock.Any(), uint64(999)).
					Return(errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name:      "database error",
			messageID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					DeleteMessageForAll(gomock.Any(), uint64(1)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.DeleteMessageForAll(ctx, tt.messageID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_ReadAllChatMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		chatID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name:   "successful read all chat messages with tracing",
			userID: 1,
			chatID: 100,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					ReadAllChatMessages(gomock.Any(), uint64(1), uint64(100)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 1,
			chatID: 100,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					ReadAllChatMessages(gomock.Any(), uint64(1), uint64(100)).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.ReadAllChatMessages(ctx, tt.userID, tt.chatID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_GetUserUnreadCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedCount uint64
		expectedError error
	}{
		{
			name:   "successful unread count with tracing",
			userID: 42,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(42)).
					Return(uint64(7), nil)
			},
			expectedCount: 7,
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 99,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserUnreadCount(gomock.Any(), uint64(99)).
					Return(uint64(0), errors.New("database connection failed"))
			},
			expectedCount: 0,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			count, err := decorator.GetUserUnreadCount(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestTraceDecorator_UpdateMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		dto           domains.UpdateMessageDTO
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockMessagesRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful update message with tracing",
			dto:  domains.UpdateMessageDTO{MessageID: 10, UserID: 1, Text: "updated text"},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{MessageID: 10, UserID: 1, Text: "updated text"}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "message not found",
			dto:  domains.UpdateMessageDTO{MessageID: 999, UserID: 1, Text: "updated text"},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{MessageID: 999, UserID: 1, Text: "updated text"}).
					Return(errors.New("message not found"))
			},
			expectedError: errors.New("message not found"),
		},
		{
			name: "database error",
			dto:  domains.UpdateMessageDTO{MessageID: 1, UserID: 2, Text: "some text"},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockMessagesRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{MessageID: 1, UserID: 2, Text: "some text"}).
					Return(errors.New("database connection failed"))
			},
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockMessagesRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := messages.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.UpdateMessage(ctx, tt.dto)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
