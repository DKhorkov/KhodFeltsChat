package chats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services/chats"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
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
		base          interfaces.ChatsService
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
			base: mockservices.NewMockChatsService(gomock.NewController(t)),
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
			base:          mockservices.NewMockChatsService(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := chats.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_GetChatMembers(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name            string
		chatID          uint64
		setupMocks      func(*mocktracing.MockProvider, *mockservices.MockChatsService, *mocktracing.MockSpan)
		expectedMembers []domains.User
		expectedError   error
	}{
		{
			name:   "successful get chat members with tracing",
			chatID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(1)).
					Return([]domains.User{
						{
							ID:        1,
							Username:  "user1",
							Email:     "user1@example.com",
							CreatedAt: now,
							UpdatedAt: now,
						},
						{
							ID:        2,
							Username:  "user2",
							Email:     "user2@example.com",
							CreatedAt: now,
							UpdatedAt: now,
						},
						{
							ID:        3,
							Username:  "user3",
							Email:     "user3@example.com",
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil)
			},
			expectedMembers: []domains.User{
				{
					ID:        1,
					Username:  "user1",
					Email:     "user1@example.com",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        2,
					Username:  "user2",
					Email:     "user2@example.com",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        3,
					Username:  "user3",
					Email:     "user3@example.com",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			expectedError: nil,
		},
		{
			name:   "get chat members - private chat with two users",
			chatID: 2,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(2)).
					Return([]domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
					}, nil)
			},
			expectedMembers: []domains.User{
				{ID: 1, Username: "user1"},
				{ID: 2, Username: "user2"},
			},
			expectedError: nil,
		},
		{
			name:   "get chat members - empty list",
			chatID: 3,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(3)).
					Return([]domains.User{}, nil)
			},
			expectedMembers: []domains.User{},
			expectedError:   nil,
		},
		{
			name:   "chat not found",
			chatID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(999)).
					Return(nil, errors.New("chat not found"))
			},
			expectedMembers: nil,
			expectedError:   errors.New("chat not found"),
		},
		{
			name:   "user not authorized to view chat members",
			chatID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(1)).
					Return(nil, errors.New("user not authorized"))
			},
			expectedMembers: nil,
			expectedError:   errors.New("user not authorized"),
		},
		{
			name:   "database error",
			chatID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetChatMembers(gomock.Any(), uint64(1)).
					Return(nil, errors.New("database connection failed"))
			},
			expectedMembers: nil,
			expectedError:   errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockChatsService(ctrl)
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

			decorator := chats.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			members, err := decorator.GetChatMembers(ctx, tt.chatID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedMembers, members)
		})
	}
}

func TestTraceDecorator_GetUserChats(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		pagination    *domains.Pagination
		setupMocks    func(*mocktracing.MockProvider, *mockservices.MockChatsService, *mocktracing.MockSpan)
		expectedChats []domains.Chat
		expectedError error
	}{
		{
			name:   "successful get user chats with tracing",
			userID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](10),
						Offset: pointers.New[uint64](0),
					}).
					Return([]domains.Chat{
						{
							ID:        1,
							Type:      domains.ChatTypePrivate,
							Title:     pointers.New[string]("Private Chat"),
							CreatedAt: now,
							UpdatedAt: now,
						},
						{
							ID:        2,
							Type:      domains.ChatTypeGroup,
							Title:     pointers.New[string]("Group Chat"),
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil)
			},
			expectedChats: []domains.Chat{
				{
					ID:        1,
					Type:      domains.ChatTypePrivate,
					Title:     pointers.New[string]("Private Chat"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        2,
					Type:      domains.ChatTypeGroup,
					Title:     pointers.New[string]("Group Chat"),
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			expectedError: nil,
		},
		{
			name:   "get user chats with custom pagination",
			userID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](20),
				Offset: pointers.New[uint64](50),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(1), &domains.Pagination{
						Limit:  pointers.New[uint64](20),
						Offset: pointers.New[uint64](50),
					}).
					Return([]domains.Chat{
						{ID: 51, Type: domains.ChatTypePrivate},
						{ID: 52, Type: domains.ChatTypeGroup},
					}, nil)
			},
			expectedChats: []domains.Chat{
				{ID: 51, Type: domains.ChatTypePrivate},
				{ID: 52, Type: domains.ChatTypeGroup},
			},
			expectedError: nil,
		},
		{
			name:   "empty chats list",
			userID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(1), gomock.Any()).
					Return([]domains.Chat{}, nil)
			},
			expectedChats: []domains.Chat{},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(999), gomock.Any()).
					Return(nil, errors.New("user not found"))
			},
			expectedChats: nil,
			expectedError: errors.New("user not found"),
		},
		{
			name:   "database error",
			userID: 1,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(1), gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			expectedChats: nil,
			expectedError: errors.New("database connection failed"),
		},
		{
			name:       "nil pagination",
			userID:     1,
			pagination: nil,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserChats(gomock.Any(), uint64(1), nil).
					Return([]domains.Chat{}, nil)
			},
			expectedChats: []domains.Chat{},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockChatsService(ctrl)
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

			decorator := chats.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			chats, err := decorator.GetUserChats(ctx, tt.userID, tt.pagination)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedChats, chats)
		})
	}
}

func TestTraceDecorator_CreateChat(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name           string
		chat           domains.Chat
		setupMocks     func(*mocktracing.MockProvider, *mockservices.MockChatsService, *mocktracing.MockSpan)
		expectedResult *domains.Chat
		expectedError  error
	}{
		{
			name: "successful create private chat with tracing",
			chat: domains.Chat{
				Type: domains.ChatTypePrivate,
				Members: []domains.User{
					{ID: 1, Username: "user1"},
					{ID: 2, Username: "user2"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					CreateChat(gomock.Any(), domains.Chat{
						Type: domains.ChatTypePrivate,
						Members: []domains.User{
							{ID: 1, Username: "user1"},
							{ID: 2, Username: "user2"},
						},
						CreatedAt: now,
						UpdatedAt: now,
					}).
					Return(&domains.Chat{
						ID:   1,
						Type: domains.ChatTypePrivate,
						Members: []domains.User{
							{ID: 1, Username: "user1"},
							{ID: 2, Username: "user2"},
						},
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedResult: &domains.Chat{
				ID:   1,
				Type: domains.ChatTypePrivate,
				Members: []domains.User{
					{ID: 1, Username: "user1"},
					{ID: 2, Username: "user2"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "successful create group chat",
			chat: domains.Chat{
				Type:      domains.ChatTypeGroup,
				Title:     pointers.New[string]("Group Chat"),
				Members:   []domains.User{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateChat(gomock.Any(), domains.Chat{
						Type:      domains.ChatTypeGroup,
						Title:     pointers.New[string]("Group Chat"),
						Members:   []domains.User{},
						CreatedAt: now,
						UpdatedAt: now,
					}).
					Return(&domains.Chat{
						ID:        2,
						Type:      domains.ChatTypeGroup,
						Title:     pointers.New[string]("Group Chat"),
						Members:   []domains.User{},
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedResult: &domains.Chat{
				ID:        2,
				Type:      domains.ChatTypeGroup,
				Title:     pointers.New[string]("Group Chat"),
				Members:   []domains.User{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name: "create chat without members",
			chat: domains.Chat{
				Type:      domains.ChatTypePrivate,
				Members:   []domains.User{},
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("private chat must have at least 2 members"))
			},
			expectedResult: nil,
			expectedError:  errors.New("private chat must have at least 2 members"),
		},
		{
			name: "duplicate chat error",
			chat: domains.Chat{
				Type: domains.ChatTypePrivate,
				Members: []domains.User{
					{ID: 1, Username: "user1"},
					{ID: 2, Username: "user2"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("chat already exists"))
			},
			expectedResult: nil,
			expectedError:  errors.New("chat already exists"),
		},
		{
			name: "database error",
			chat: domains.Chat{
				Type:  domains.ChatTypeGroup,
				Title: pointers.New[string]("New Chat"),
				Members: []domains.User{
					{ID: 1},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			expectedResult: nil,
			expectedError:  errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockChatsService(ctrl)
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

			decorator := chats.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			result, err := decorator.CreateChat(ctx, tt.chat)

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

func TestTraceDecorator_PrivateChatExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		members        []domains.User
		setupMocks     func(*mocktracing.MockProvider, *mockservices.MockChatsService, *mocktracing.MockSpan)
		expectedExists bool
		expectedError  error
	}{
		{
			name: "private chat exists",
			members: []domains.User{
				{ID: 1, Username: "user1"},
				{ID: 2, Username: "user2"},
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					PrivateChatExists(gomock.Any(), []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
					}).
					Return(true, nil)
			},
			expectedExists: true,
			expectedError:  nil,
		},
		{
			name: "private chat does not exist",
			members: []domains.User{
				{ID: 1, Username: "user1"},
				{ID: 3, Username: "user3"},
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					PrivateChatExists(gomock.Any(), []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 3, Username: "user3"},
					}).
					Return(false, nil)
			},
			expectedExists: false,
			expectedError:  nil,
		},
		{
			name:    "empty members list",
			members: []domains.User{},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					PrivateChatExists(gomock.Any(), []domains.User{}).
					Return(false, errors.New("invalid members count"))
			},
			expectedExists: false,
			expectedError:  errors.New("invalid members count"),
		},
		{
			name: "too many members for private chat",
			members: []domains.User{
				{ID: 1, Username: "user1"},
				{ID: 2, Username: "user2"},
				{ID: 3, Username: "user3"},
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					PrivateChatExists(gomock.Any(), []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
						{ID: 3, Username: "user3"},
					}).
					Return(false, errors.New("private chat can only have 2 members"))
			},
			expectedExists: false,
			expectedError:  errors.New("private chat can only have 2 members"),
		},
		{
			name: "database error",
			members: []domains.User{
				{ID: 1, Username: "user1"},
				{ID: 2, Username: "user2"},
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockservices.MockChatsService,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					PrivateChatExists(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database connection failed"))
			},
			expectedExists: false,
			expectedError:  errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockservices.NewMockChatsService(ctrl)
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

			decorator := chats.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			exists, err := decorator.PrivateChatExists(ctx, tt.members)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}
