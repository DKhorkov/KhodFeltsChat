package users_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/usecases/users"
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
		base          interfaces.UsersUseCases
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
			base: mockusecases.NewMockUsersUseCases(gomock.NewController(t)),
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
			base:          mockusecases.NewMockUsersUseCases(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := users.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_GetUsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filters       *domains.UsersFilters
		pagination    *domains.Pagination
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockUsersUseCases, *mocktracing.MockSpan)
		expectedUsers []domains.User
		expectedError error
	}{
		{
			name:    "successful get users with tracing",
			filters: &domains.UsersFilters{},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{}, &domains.Pagination{Limit: pointers.New[uint64](10), Offset: pointers.New[uint64](0)}).
					Return([]domains.User{
						{ID: 1, Username: "user1", Email: "user1@example.com"},
						{ID: 2, Username: "user2", Email: "user2@example.com"},
					}, nil)
			},
			expectedUsers: []domains.User{
				{ID: 1, Username: "user1", Email: "user1@example.com"},
				{ID: 2, Username: "user2", Email: "user2@example.com"},
			},
			expectedError: nil,
		},
		{
			name:    "get users with empty filters",
			filters: nil,
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](5),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), nil, &domains.Pagination{Limit: pointers.New[uint64](5), Offset: pointers.New[uint64](0)}).
					Return([]domains.User{}, nil)
			},
			expectedUsers: []domains.User{},
			expectedError: nil,
		},
		{
			name:    "get users with pagination",
			filters: &domains.UsersFilters{},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](20),
				Offset: pointers.New[uint64](50),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{}, &domains.Pagination{Limit: pointers.New[uint64](20), Offset: pointers.New[uint64](50)}).
					Return([]domains.User{
						{ID: 51, Username: "user51"},
						{ID: 52, Username: "user52"},
					}, nil)
			},
			expectedUsers: []domains.User{
				{ID: 51, Username: "user51"},
				{ID: 52, Username: "user52"},
			},
			expectedError: nil,
		},
		{
			name:    "base returns error",
			filters: &domains.UsersFilters{},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{}, &domains.Pagination{Limit: pointers.New[uint64](10), Offset: pointers.New[uint64](0)}).
					Return(nil, errors.New("database error"))
			},
			expectedUsers: nil,
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockUsersUseCases(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			users, err := decorator.GetUsers(ctx, tt.filters, tt.pagination)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedUsers, users)
		})
	}
}

func TestTraceDecorator_GetUserByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockUsersUseCases, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name:   "successful get user by id",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(&domains.User{
						ID:       1,
						Username: "john_doe",
						Email:    "john@example.com",
					}, nil)
			},
			expectedUser: &domains.User{
				ID:       1,
				Username: "john_doe",
				Email:    "john@example.com",
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByID(gomock.Any(), uint64(999)).
					Return(nil, errors.New("user not found"))
			},
			expectedUser:  nil,
			expectedError: errors.New("user not found"),
		},
		{
			name:   "zero user id",
			userID: 0,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByID(gomock.Any(), uint64(0)).
					Return(nil, errors.New("invalid user id"))
			},
			expectedUser:  nil,
			expectedError: errors.New("invalid user id"),
		},
		{
			name:   "database error",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(nil, errors.New("connection timeout"))
			},
			expectedUser:  nil,
			expectedError: errors.New("connection timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockUsersUseCases(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			user, err := decorator.GetUserByID(ctx, tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedUser, user)
		})
	}
}

func TestTraceDecorator_UpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userData      domains.UpdateUserDTO
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockUsersUseCases, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name: "successful update user",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: pointers.New("updated_user"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       1,
						Username: pointers.New("updated_user"),
					}).
					Return(&domains.User{
						ID:       1,
						Username: "updated_user",
						Email:    "updated@example.com",
					}, nil)
			},
			expectedUser: &domains.User{
				ID:       1,
				Username: "updated_user",
				Email:    "updated@example.com",
			},
			expectedError: nil,
		},
		{
			name: "update user with only username",
			userData: domains.UpdateUserDTO{
				ID:       3,
				Username: pointers.New("new_username"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       3,
						Username: pointers.New("new_username"),
					}).
					Return(&domains.User{
						ID:       3,
						Username: "new_username",
						Email:    "user3@example.com",
					}, nil)
			},
			expectedUser: &domains.User{
				ID:       3,
				Username: "new_username",
				Email:    "user3@example.com",
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			userData: domains.UpdateUserDTO{
				ID:       999,
				Username: pointers.New("nonexistent"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       999,
						Username: pointers.New("nonexistent"),
					}).
					Return(nil, errors.New("user not found"))
			},
			expectedUser:  nil,
			expectedError: errors.New("user not found"),
		},
		{
			name: "duplicate email error",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: pointers.New("existing@example.com"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       1,
						Username: pointers.New("existing@example.com"),
					}).
					Return(nil, errors.New("email already exists"))
			},
			expectedUser:  nil,
			expectedError: errors.New("email already exists"),
		},
		{
			name: "invalid user id",
			userData: domains.UpdateUserDTO{
				ID:       0,
				Username: pointers.New("invalid"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       0,
						Username: pointers.New("invalid"),
					}).
					Return(nil, errors.New("invalid user id"))
			},
			expectedUser:  nil,
			expectedError: errors.New("invalid user id"),
		},
		{
			name: "database error",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: pointers.New("test"),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockUsersUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), domains.UpdateUserDTO{
						ID:       1,
						Username: pointers.New("test"),
					}).
					Return(nil, errors.New("database connection failed"))
			},
			expectedUser:  nil,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockUsersUseCases(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			user, err := decorator.UpdateUser(ctx, tt.userData)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedUser, user)
		})
	}
}
