package messages_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/usecases/messages"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// newMockReactionsUseCases возвращает мок с открытыми AnyTimes-ожиданиями
// на AttachReactions / AttachReaction — они возвращают вход без изменений,
// что позволяет тестам messages usecase не заботиться о реакциях.
func newMockReactionsUseCases(ctrl *gomock.Controller) *mockusecases.MockReactionsUseCases {
	m := mockusecases.NewMockReactionsUseCases(ctrl)
	m.EXPECT().
		AttachReactions(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msgs []domains.Message) ([]domains.Message, error) {
			return msgs, nil
		}).
		AnyTimes()
	m.EXPECT().
		AttachReaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, msg *domains.Message) (*domains.Message, error) {
			return msg, nil
		}).
		AnyTimes()

	return m
}

func TestUseCases_SaveMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx     context.Context
		message domains.Message
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Message
		wantErr bool
		err     error
	}{
		{
			name: "successfully save message",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(domains.Message{})).
						Return(&domains.Message{
							ID:     1,
							ChatID: 100,
							Text:   "Hello, world!",
							Sender: domains.User{ID: 1},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				message: domains.Message{
					ChatID: 100,
					Text:   "Hello, world!",
					Sender: domains.User{ID: 1},
				},
			},
			want: &domains.Message{
				ID:     1,
				ChatID: 100,
				Text:   "Hello, world!",
				Sender: domains.User{ID: 1},
			},
			wantErr: false,
		},
		{
			name: "save message with service error",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(domains.Message{})).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				message: domains.Message{
					ChatID: 100,
					Text:   "Hello, world!",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.SaveMessage(tt.args.ctx, tt.args.message)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_GetChatMessages(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx        context.Context
		userID     uint64
		chatID     uint64
		pagination *domains.Pagination
	}

	testUser := &domains.User{ID: 1, Username: "testuser"}
	chatMembers := []domains.User{
		{ID: 1, Username: "testuser"},
		{ID: 2, Username: "otheruser"},
	}
	testMessages := []domains.Message{
		{ID: 1, ChatID: 100, Text: "Message 1", Sender: domains.User{ID: 1}},
		{ID: 2, ChatID: 100, Text: "Message 2", Sender: domains.User{ID: 2}},
	}

	limit := uint64(10)
	offset := uint64(0)
	paginationWithValues := &domains.Pagination{
		Limit:  &limit,
		Offset: &offset,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.Message
		wantErr bool
		err     error
	}{
		{
			name: "successfully get chat messages with pagination",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), paginationWithValues).
						Return(testMessages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: paginationWithValues,
			},
			want:    testMessages,
			wantErr: false,
		},
		{
			name: "successfully get chat messages without pagination (nil)",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), (*domains.Pagination)(nil)).
						Return(testMessages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: nil,
			},
			want:    testMessages,
			wantErr: false,
		},
		{
			name: "user not found",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(999)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     999,
				chatID:     100,
				pagination: nil,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "chat members retrieval error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(nil, errors.New("chat not found"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: nil,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("chat not found"),
		},
		{
			name: "user is not chat member",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(3)).
						Return(&domains.User{ID: 3, Username: "stranger"}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(3)).
						Return(chatMembers, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     3,
				chatID:     100,
				pagination: nil,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user is not a chat member"),
		},
		{
			name: "get messages service error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), gomock.Any()).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: nil,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "empty chat members list",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return([]domains.User{}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: nil,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user is not a chat member"),
		},
		{
			name: "user is the only member in chat",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101), uint64(1)).
						Return([]domains.User{{ID: 1, Username: "testuser"}}, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(101), gomock.Any()).
						Return([]domains.Message{
							{ID: 1, ChatID: 101, Text: "Self message", Sender: domains.User{ID: 1}},
						}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     101,
				pagination: nil,
			},
			want: []domains.Message{
				{ID: 1, ChatID: 101, Text: "Self message", Sender: domains.User{ID: 1}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.GetChatMessages(
				tt.args.ctx,
				tt.args.userID,
				tt.args.chatID,
				tt.args.pagination,
			)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_GetChatMessages_WithPaginationVariations(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx        context.Context
		userID     uint64
		chatID     uint64
		pagination *domains.Pagination
	}

	testUser := &domains.User{ID: 1, Username: "testuser"}
	chatMembers := []domains.User{{ID: 1}, {ID: 2}}
	testMessages := []domains.Message{{ID: 1}}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.Message
		wantErr bool
	}{
		{
			name: "pagination with only limit",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().GetUserByID(gomock.Any(), uint64(1)).Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					limit := uint64(10)
					expectedPagination := &domains.Pagination{Limit: &limit, Offset: nil}
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), expectedPagination).
						Return(testMessages, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				chatID: 100,
				pagination: &domains.Pagination{
					Limit: func() *uint64 {
						l := uint64(10)

						return &l
					}(),
				},
			},
			want: testMessages,
		},
		{
			name: "pagination with only offset",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().GetUserByID(gomock.Any(), uint64(1)).Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					offset := uint64(20)
					expectedPagination := &domains.Pagination{Limit: nil, Offset: &offset}
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), expectedPagination).
						Return(testMessages, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				chatID: 100,
				pagination: &domains.Pagination{
					Offset: func() *uint64 {
						o := uint64(20)

						return &o
					}(),
				},
			},
			want: testMessages,
		},
		{
			name: "pagination with zero values",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().GetUserByID(gomock.Any(), uint64(1)).Return(testUser, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(1)).
						Return(chatMembers, nil)
				},
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					zero := uint64(0)
					expectedPagination := &domains.Pagination{Limit: &zero, Offset: &zero}
					ms.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), expectedPagination).
						Return(testMessages, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				chatID: 100,
				pagination: &domains.Pagination{
					Limit: func() *uint64 {
						l := uint64(0)

						return &l
					}(),
					Offset: func() *uint64 {
						o := uint64(0)

						return &o
					}(),
				},
			},
			want: testMessages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.GetChatMessages(
				tt.args.ctx,
				tt.args.userID,
				tt.args.chatID,
				tt.args.pagination,
			)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_GetMessageByID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx       context.Context
		userID    uint64
		messageID uint64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Message
		wantErr bool
		err     error
	}{
		{
			name: "successfully get message by id",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Hello, world!",
							Sender: domains.User{ID: 1},
						}, nil)
				},
			},
			args: args{
				ctx:       context.Background(),
				userID:    1,
				messageID: 10,
			},
			want: &domains.Message{
				ID:     10,
				ChatID: 100,
				Text:   "Hello, world!",
				Sender: domains.User{ID: 1},
			},
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
						Return(nil, errors.New("message not found"))
				},
			},
			args: args{
				ctx:       context.Background(),
				userID:    1,
				messageID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("message not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.GetMessageByID(tt.args.ctx, tt.args.userID, tt.args.messageID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_GetUserUnreadCount(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    uint64
		wantErr bool
		err     error
	}{
		{
			name: "successfully get user unread count",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetUserUnreadCount(gomock.Any(), uint64(1)).
						Return(uint64(42), nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
			},
			want:    42,
			wantErr: false,
		},
		{
			name: "zero unread messages",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetUserUnreadCount(gomock.Any(), uint64(2)).
						Return(uint64(0), nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 2,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetUserUnreadCount(gomock.Any(), uint64(999)).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			want:    0,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.GetUserUnreadCount(tt.args.ctx, tt.args.userID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_DeleteMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx context.Context
		dto domains.DeleteMessageDTO
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully delete for self",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
							MessageID: 10,
							UserID:    1,
							ForAll:    false,
						}).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.DeleteMessageDTO{
					MessageID: 10,
					UserID:    1,
					ForAll:    false,
				},
			},
			wantErr: false,
		},
		{
			name: "successfully delete for all (user is author)",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Hello",
							Sender: domains.User{ID: 1},
						}, nil)

					ms.EXPECT().
						DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
							MessageID: 10,
							UserID:    1,
							ForAll:    true,
						}).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.DeleteMessageDTO{
					MessageID: 10,
					UserID:    1,
					ForAll:    true,
				},
			},
			wantErr: false,
		},
		{
			name: "delete for all - message not found",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
						Return(nil, errors.New("message not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.DeleteMessageDTO{
					MessageID: 999,
					UserID:    1,
					ForAll:    true,
				},
			},
			wantErr: true,
			err:     customerrors.ErrMessageNotFound,
		},
		{
			name: "delete for all - not message author",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(2), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Hello",
							Sender: domains.User{ID: 1},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.DeleteMessageDTO{
					MessageID: 10,
					UserID:    2,
					ForAll:    true,
				},
			},
			wantErr: true,
			err:     customerrors.ErrNotMessageAuthor,
		},
		{
			name: "service DeleteMessage error",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
							MessageID: 10,
							UserID:    1,
							ForAll:    false,
						}).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.DeleteMessageDTO{
					MessageID: 10,
					UserID:    1,
					ForAll:    false,
				},
			},
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			err := uc.DeleteMessage(tt.args.ctx, tt.args.dto)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUseCases_UpdateMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockMessagesService func(*mockservices.MockMessagesService)
		mockUsersService    func(*mockservices.MockUsersService)
		mockChatsService    func(*mockservices.MockChatsService)
	}

	type args struct {
		ctx context.Context
		dto domains.UpdateMessageDTO
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Message
		wantErr bool
		err     error
	}{
		{
			name: "successfully update message",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Old text",
							Sender: domains.User{ID: 1},
						}, nil)

					ms.EXPECT().
						UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
							MessageID: 10,
							UserID:    1,
							Text:      "New text",
						}).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "New text",
							Sender: domains.User{ID: 1},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.UpdateMessageDTO{
					MessageID: 10,
					UserID:    1,
					Text:      "New text",
				},
			},
			want: &domains.Message{
				ID:     10,
				ChatID: 100,
				Text:   "New text",
				Sender: domains.User{ID: 1},
			},
			wantErr: false,
		},
		{
			name: "message not found",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(999)).
						Return(nil, errors.New("message not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.UpdateMessageDTO{
					MessageID: 999,
					UserID:    1,
					Text:      "New text",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrMessageNotFound,
		},
		{
			name: "not message author",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(2), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Hello",
							Sender: domains.User{ID: 1},
						}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.UpdateMessageDTO{
					MessageID: 10,
					UserID:    2,
					Text:      "New text",
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrNotMessageAuthor,
		},
		{
			name: "service UpdateMessage error",
			fields: fields{
				mockMessagesService: func(ms *mockservices.MockMessagesService) {
					ms.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(&domains.Message{
							ID:     10,
							ChatID: 100,
							Text:   "Hello",
							Sender: domains.User{ID: 1},
						}, nil)

					ms.EXPECT().
						UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
							MessageID: 10,
							UserID:    1,
							Text:      "New text",
						}).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: domains.UpdateMessageDTO{
					MessageID: 10,
					UserID:    1,
					Text:      "New text",
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockMessagesService := mockservices.NewMockMessagesService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)
			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockReactionsUseCases := newMockReactionsUseCases(ctrl)

			if tt.fields.mockMessagesService != nil {
				tt.fields.mockMessagesService(mockMessagesService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			uc := messages.New(
				mockMessagesService,
				mockChatsService,
				mockUsersService,
				mockReactionsUseCases,
			)

			// Act
			got, err := uc.UpdateMessage(tt.args.ctx, tt.args.dto)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_GetChatMessages_AttachesReactions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockMessagesService := mockservices.NewMockMessagesService(ctrl)
	mockUsersService := mockservices.NewMockUsersService(ctrl)
	mockChatsService := mockservices.NewMockChatsService(ctrl)
	mockReactions := mockusecases.NewMockReactionsUseCases(ctrl)

	ctx := context.Background()
	userID := uint64(1)
	chatID := uint64(100)
	svcMsgs := []domains.Message{{ID: 10, ChatID: chatID}, {ID: 20, ChatID: chatID}}
	withReactions := []domains.Message{
		{ID: 10, ChatID: chatID, Reactions: []domains.MessageReactionSummary{
			{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{7}},
		}},
		{ID: 20, ChatID: chatID},
	}

	mockUsersService.EXPECT().
		GetUserByID(gomock.Any(), userID).
		Return(&domains.User{ID: userID}, nil)
	mockChatsService.EXPECT().GetChatMembers(gomock.Any(), chatID, userID).
		Return([]domains.User{{ID: userID}}, nil)
	mockMessagesService.EXPECT().
		GetChatMessages(gomock.Any(), userID, chatID, gomock.Any()).
		Return(svcMsgs, nil)
	mockReactions.EXPECT().AttachReactions(gomock.Any(), svcMsgs).Return(withReactions, nil)

	uc := messages.New(mockMessagesService, mockChatsService, mockUsersService, mockReactions)

	got, err := uc.GetChatMessages(ctx, userID, chatID, nil)
	assert.NoError(t, err)
	assert.Equal(t, withReactions, got)
}

func TestUseCases_GetMessageByID_AttachesReaction(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockMessagesService := mockservices.NewMockMessagesService(ctrl)
	mockUsersService := mockservices.NewMockUsersService(ctrl)
	mockChatsService := mockservices.NewMockChatsService(ctrl)
	mockReactions := mockusecases.NewMockReactionsUseCases(ctrl)

	ctx := context.Background()
	svcMsg := &domains.Message{ID: 10, ChatID: 100}
	withReaction := &domains.Message{
		ID:     10,
		ChatID: 100,
		Reactions: []domains.MessageReactionSummary{
			{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{7}},
		},
	}

	mockMessagesService.EXPECT().
		GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
		Return(svcMsg, nil)
	mockReactions.EXPECT().AttachReaction(gomock.Any(), svcMsg).Return(withReaction, nil)

	uc := messages.New(mockMessagesService, mockChatsService, mockUsersService, mockReactions)

	got, err := uc.GetMessageByID(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, withReaction, got)
}
