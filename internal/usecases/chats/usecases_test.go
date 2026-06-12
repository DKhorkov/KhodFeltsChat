package chats_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/usecases/chats"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_GetChatMembers(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockChatsService func(*mockservices.MockChatsService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx    context.Context
		chatID uint64
	}

	expectedMembers := []domains.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.User
		wantErr bool
		err     error
	}{
		{
			name: "successfully get chat members",
			fields: fields{
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(0)).
						Return(expectedMembers, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    expectedMembers,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100), uint64(0)).
						Return(nil, errors.New("chat not found"))
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("chat not found"),
		},
		{
			name: "empty chat members list",
			fields: fields{
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101), uint64(0)).
						Return([]domains.User{}, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 101,
			},
			want:    []domains.User{},
			wantErr: false,
		},
		{
			name: "chat with single member",
			fields: fields{
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetChatMembers(gomock.Any(), uint64(102), uint64(0)).
						Return([]domains.User{{ID: 1, Username: "singleuser"}}, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 102,
			},
			want:    []domains.User{{ID: 1, Username: "singleuser"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			uc := chats.New(mockChatsService, mockUsersService)

			// Act
			got, err := uc.GetChatMembers(tt.args.ctx, tt.args.chatID, 0)

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

func TestUseCases_GetUserChats(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockChatsService func(*mockservices.MockChatsService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx        context.Context
		userID     uint64
		pagination *domains.Pagination
	}

	user := &domains.User{ID: 1, Username: "testuser"}
	expectedChats := []domains.Chat{
		{ID: 1, Title: pointers.New("Chat 1"), Type: domains.ChatTypeGroup},
		{ID: 2, Title: pointers.New("Chat 2"), Type: domains.ChatTypePrivate},
	}

	limit := uint64(10)
	offset := uint64(0)
	pagination := &domains.Pagination{Limit: &limit, Offset: &offset}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.Chat
		wantErr bool
		err     error
	}{
		{
			name: "successfully get user chats with pagination",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(user, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(expectedChats, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want:    expectedChats,
			wantErr: false,
		},
		{
			name: "successfully get user chats without pagination",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(user, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), (*domains.Pagination)(nil)).
						Return(expectedChats, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: nil,
			},
			want:    expectedChats,
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
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "chats service returns error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(user, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "empty chat list returned",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2, Username: "newuser"}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						GetUserChats(gomock.Any(), uint64(2), pagination).
						Return([]domains.Chat{}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     2,
				pagination: pagination,
			},
			want:    []domains.Chat{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			uc := chats.New(mockChatsService, mockUsersService)

			// Act
			got, err := uc.GetUserChats(tt.args.ctx, tt.args.userID, tt.args.pagination)

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

func TestUseCases_CreateChat(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockChatsService func(*mockservices.MockChatsService)
		mockUsersService func(*mockservices.MockUsersService)
	}

	type args struct {
		ctx  context.Context
		chat domains.Chat
	}

	title := "Test Chat"
	description := "Test Description"
	members := []domains.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
	}

	validGroupChat := domains.Chat{
		ID:          0, // ID будет установлен сервисом
		Title:       &title,
		Description: &description,
		Type:        domains.ChatTypeGroup,
		Members:     members,
	}

	validPrivateChat := domains.Chat{
		Type:    domains.ChatTypePrivate,
		Members: []domains.User{{ID: 1}, {ID: 2}},
	}

	createdGroupChat := validGroupChat
	createdGroupChat.ID = 100

	createdPrivateChat := validPrivateChat
	createdPrivateChat.ID = 101

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Chat
		wantErr bool
		err     error
	}{
		{
			name: "successfully create group chat",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					// Проверяем обоих участников
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					// Для группового чата не проверяем существование приватного чата
					cs.EXPECT().
						CreateChat(gomock.Any(), gomock.AssignableToTypeOf(domains.Chat{})).
						Return(&createdGroupChat, nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: validGroupChat,
			},
			want:    &createdGroupChat,
			wantErr: false,
		},
		{
			name: "successfully create private chat",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						PrivateChatExists(gomock.Any(), gomock.AssignableToTypeOf([]domains.User{})).
						Return(false, nil)
					cs.EXPECT().
						CreateChat(gomock.Any(), gomock.AssignableToTypeOf(domains.Chat{})).
						Return(&createdPrivateChat, nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: validPrivateChat,
			},
			want:    &createdPrivateChat,
			wantErr: false,
		},
		{
			name: "invalid chat - wrong type",
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type:    "invalid_type",
					Members: []domains.User{{ID: 1}},
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrInvalidChat,
		},
		{
			name: "invalid chat - no members",
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type:    domains.ChatTypeGroup,
					Members: []domains.User{},
				},
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrInvalidChat,
		},
		{
			name: "user not found when validating members",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().GetUserByID(gomock.Any(), uint64(1)).
						Return(nil, errors.New("user not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type:    domains.ChatTypeGroup,
					Members: []domains.User{{ID: 1}, {ID: 2}},
				},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("user not found"),
		},
		{
			name: "private chat already exists",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						PrivateChatExists(gomock.Any(), gomock.AssignableToTypeOf([]domains.User{})).
						Return(true, nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: validPrivateChat,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatAlreadyExists,
		},
		{
			name: "private chat exists check returns error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						PrivateChatExists(gomock.Any(), gomock.AssignableToTypeOf([]domains.User{})).
						Return(false, errors.New("database error"))
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: validPrivateChat,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "create chat service returns error",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(2)).
						Return(&domains.User{ID: 2}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						CreateChat(gomock.Any(), gomock.AssignableToTypeOf(domains.Chat{})).
						Return(nil, errors.New("create error"))
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: validGroupChat,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("create error"),
		},
		{
			name: "chat with single member (valid)",
			fields: fields{
				mockUsersService: func(us *mockservices.MockUsersService) {
					us.EXPECT().
						GetUserByID(gomock.Any(), uint64(1)).
						Return(&domains.User{ID: 1}, nil)
				},
				mockChatsService: func(cs *mockservices.MockChatsService) {
					cs.EXPECT().
						CreateChat(gomock.Any(), gomock.AssignableToTypeOf(domains.Chat{})).
						Return(&domains.Chat{ID: 103, Type: domains.ChatTypeGroup, Members: []domains.User{{ID: 1}}}, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type:    domains.ChatTypeGroup,
					Members: []domains.User{{ID: 1}},
				},
			},
			want: &domains.Chat{
				ID:      103,
				Type:    domains.ChatTypeGroup,
				Members: []domains.User{{ID: 1}},
			},
			wantErr: false,
		},
		{
			name: "private chat with single member  validation fail",
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type:    domains.ChatTypePrivate,
					Members: []domains.User{{ID: 1}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockChatsService := mockservices.NewMockChatsService(ctrl)
			mockUsersService := mockservices.NewMockUsersService(ctrl)

			if tt.fields.mockChatsService != nil {
				tt.fields.mockChatsService(mockChatsService)
			}

			if tt.fields.mockUsersService != nil {
				tt.fields.mockUsersService(mockUsersService)
			}

			uc := chats.New(mockChatsService, mockUsersService)

			// Act
			got, err := uc.CreateChat(tt.args.ctx, tt.args.chat)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrInvalidChat) ||
						errors.Is(tt.err, customerrors.ErrChatAlreadyExists) {
						assert.ErrorIs(t, err, tt.err)
					} else {
						assert.Contains(t, err.Error(), tt.err.Error())
					}
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
