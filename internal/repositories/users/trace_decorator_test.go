package users_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories/users"
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
		base          interfaces.UsersRepository
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
			base: mockrepositories.NewMockUsersRepository(gomock.NewController(t)),
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
			base:          mockrepositories.NewMockUsersRepository(gomock.NewController(t)),
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

func TestTraceDecorator_GetUserByID(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		userID        uint64
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockUsersRepository, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name:   "successful get user by id with tracing",
			userID: 1,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
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
						ID:        1,
						Username:  "john_doe",
						Email:     "john@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedUser: &domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
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
			name:   "invalid user id",
			userID: 0,
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockUsersRepository(ctrl)
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

func TestTraceDecorator_GetUserByUsername(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		username      string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockUsersRepository, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name:     "successful get user by username with tracing",
			username: "john_doe",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUserByUsername(gomock.Any(), "john_doe").
					Return(&domains.User{
						ID:        1,
						Username:  "john_doe",
						Email:     "john@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedUser: &domains.User{
				ID:        1,
				Username:  "john_doe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:     "user not found by username",
			username: "nonexistent",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByUsername(gomock.Any(), "nonexistent").
					Return(nil, errors.New("user not found"))
			},
			expectedUser:  nil,
			expectedError: errors.New("user not found"),
		},
		{
			name:     "empty username",
			username: "",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByUsername(gomock.Any(), "").
					Return(nil, errors.New("username is required"))
			},
			expectedUser:  nil,
			expectedError: errors.New("username is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockUsersRepository(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			user, err := decorator.GetUserByUsername(ctx, tt.username)

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

func TestTraceDecorator_GetUserByEmail(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		email         string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockUsersRepository, *mocktracing.MockSpan)
		expectedUser  *domains.User
		expectedError error
	}{
		{
			name:  "successful get user by email with tracing",
			email: "user@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUserByEmail(gomock.Any(), "user@example.com").
					Return(&domains.User{
						ID:        1,
						Username:  "user",
						Email:     "user@example.com",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedUser: &domains.User{
				ID:        1,
				Username:  "user",
				Email:     "user@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedError: nil,
		},
		{
			name:  "user not found by email",
			email: "nonexistent@example.com",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByEmail(gomock.Any(), "nonexistent@example.com").
					Return(nil, errors.New("user not found"))
			},
			expectedUser:  nil,
			expectedError: errors.New("user not found"),
		},
		{
			name:  "invalid email format",
			email: "invalid-email",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUserByEmail(gomock.Any(), "invalid-email").
					Return(nil, errors.New("invalid email format"))
			},
			expectedUser:  nil,
			expectedError: errors.New("invalid email format"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockUsersRepository(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			user, err := decorator.GetUserByEmail(ctx, tt.email)

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

func TestTraceDecorator_GetUsers(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		filters       *domains.UsersFilters
		pagination    *domains.Pagination
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockUsersRepository, *mocktracing.MockSpan)
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
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{}, &domains.Pagination{
						Limit:  pointers.New[uint64](10),
						Offset: pointers.New[uint64](0),
					}).
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
					}, nil)
			},
			expectedUsers: []domains.User{
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
			},
			expectedError: nil,
		},
		{
			name:    "get users with filters",
			filters: &domains.UsersFilters{Username: pointers.New[string]("john")},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{Username: pointers.New[string]("john")}, gomock.Any()).
					Return([]domains.User{
						{ID: 1, Username: "john_doe"},
					}, nil)
			},
			expectedUsers: []domains.User{
				{ID: 1, Username: "john_doe"},
			},
			expectedError: nil,
		},
		{
			name:    "empty users list",
			filters: &domains.UsersFilters{},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]domains.User{}, nil)
			},
			expectedUsers: []domains.User{},
			expectedError: nil,
		},
		{
			name:    "database error",
			filters: &domains.UsersFilters{},
			pagination: &domains.Pagination{
				Limit:  pointers.New[uint64](10),
				Offset: pointers.New[uint64](0),
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					GetUsers(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			expectedUsers: nil,
			expectedError: errors.New("database connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockUsersRepository(ctrl)
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

func TestTraceDecorator_UpdateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		userData      domains.UpdateUserDTO
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockUsersRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful update user with tracing",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: "updated_user",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
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
						Username: "updated_user",
					}).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			userData: domains.UpdateUserDTO{
				ID:       999,
				Username: "nonexistent",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("user not found"))
			},
			expectedError: errors.New("user not found"),
		},
		{
			name: "duplicate username error",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: "existing_username",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Return(errors.New("username already exists"))
			},
			expectedError: errors.New("username already exists"),
		},
		{
			name: "database error",
			userData: domains.UpdateUserDTO{
				ID:       1,
				Username: "test",
			},
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockUsersRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
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
			mockBase := mockrepositories.NewMockUsersRepository(ctrl)
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

			decorator := users.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			ctx := context.Background()
			err := decorator.UpdateUser(ctx, tt.userData)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
