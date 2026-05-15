package messages_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/usecases/messages"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(100)).
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
						GetChatMembers(gomock.Any(), uint64(101)).
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
					cs.EXPECT().GetChatMembers(gomock.Any(), uint64(100)).Return(chatMembers, nil)
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
					cs.EXPECT().GetChatMembers(gomock.Any(), uint64(100)).Return(chatMembers, nil)
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
					cs.EXPECT().GetChatMembers(gomock.Any(), uint64(100)).Return(chatMembers, nil)
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
