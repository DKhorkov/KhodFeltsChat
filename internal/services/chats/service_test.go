package chats_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/chats"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_GetChatMembers(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx    context.Context
		chatID uint64
	}

	chat := &domains.Chat{ID: 100, Title: pointers.New("Test Chat")}
	members := []domains.User{
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
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(100)).
						Return(chat, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(members, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    members,
			wantErr: false,
		},
		{
			name: "chat not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(999)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatNotFound,
		},
		{
			name: "error getting chat",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(100)).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatNotFound,
		},
		{
			name: "error getting chat members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(100)).
						Return(chat, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "empty chat members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(101)).
						Return(&domains.Chat{ID: 101}, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101)).
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockChatsRepo := mockrepositories.NewMockChatsRepository(ctrl)
			mockMessagesRepo := mockrepositories.NewMockMessagesRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockChatsRepository != nil {
				tt.fields.mockChatsRepository(mockChatsRepo)
			}

			if tt.fields.mockMessagesRepository != nil {
				tt.fields.mockMessagesRepository(mockMessagesRepo)
			}

			newChatsRepoFunc := func(_ pg.Transaction) interfaces.ChatsRepository {
				return mockChatsRepo
			}

			newMessagesRepoFunc := func(_ pg.Transaction) interfaces.MessagesRepository {
				return mockMessagesRepo
			}

			s := service.New(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := s.GetChatMembers(tt.args.ctx, tt.args.chatID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrChatNotFound) {
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

func TestService_GetChatByID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW             func(*mockuow.MockUnitOfWork)
		mockChatsRepository func(*mockrepositories.MockChatsRepository)
	}

	type args struct {
		ctx    context.Context
		chatID uint64
	}

	chat := &domains.Chat{ID: 100, Title: pointers.New("Test Chat")}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Chat
		wantErr bool
		err     error
	}{
		{
			name: "successfully get chat by id",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(100)).
						Return(chat, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    chat,
			wantErr: false,
		},
		{
			name: "chat not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(999)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatNotFound,
		},
		{
			name: "database error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(100)).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				chatID: 100,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockChatsRepo := mockrepositories.NewMockChatsRepository(ctrl)
			mockMessagesRepo := mockrepositories.NewMockMessagesRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockChatsRepository != nil {
				tt.fields.mockChatsRepository(mockChatsRepo)
			}

			newChatsRepoFunc := func(_ pg.Transaction) interfaces.ChatsRepository {
				return mockChatsRepo
			}

			newMessagesRepoFunc := func(_ pg.Transaction) interfaces.MessagesRepository {
				return mockMessagesRepo
			}

			s := service.New(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := s.GetChatByID(tt.args.ctx, tt.args.chatID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					if errors.Is(tt.err, customerrors.ErrChatNotFound) {
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

func TestService_GetUserChats(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx        context.Context
		userID     uint64
		pagination *domains.Pagination
	}

	chats := []domains.Chat{
		{ID: 100, Title: pointers.New("Chat 1")},
		{ID: 101, Title: pointers.New("Chat 2")},
	}

	members1 := []domains.User{{ID: 1, Username: "user1"}, {ID: 2, Username: "user2"}}
	members2 := []domains.User{{ID: 1, Username: "user1"}, {ID: 3, Username: "user3"}}

	messages1 := []domains.Message{{ID: 1, Text: "Last message 1"}}
	messages2 := []domains.Message{{ID: 2, Text: "Last message 2"}}

	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](10),
		Offset: pointers.New[uint64](0),
	}

	messagesPagination := &domains.Pagination{
		Limit: pointers.New[uint64](1),
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domains.Chat
		wantErr bool
		err     error
	}{
		{
			name: "successfully get user chats with members and messages",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					// Get user chats
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(chats, nil)

					// Get members for each chat
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(members1, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101)).
						Return(members2, nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					// Get last message for each chat
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), messagesPagination).
						Return(messages1, nil)

					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(101), messagesPagination).
						Return(messages2, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want: []domains.Chat{
				{ID: 100, Title: pointers.New("Chat 1"), Members: members1, Messages: messages1},
				{ID: 101, Title: pointers.New("Chat 2"), Members: members2, Messages: messages2},
			},
			wantErr: false,
		},
		{
			name: "successfully get user chats without pagination",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), (*domains.Pagination)(nil)).
						Return(chats, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(members1, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101)).
						Return(members2, nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), messagesPagination).
						Return(messages1, nil)

					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(101), messagesPagination).
						Return(messages2, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: nil,
			},
			want: []domains.Chat{
				{ID: 100, Title: pointers.New("Chat 1"), Members: members1, Messages: messages1},
				{ID: 101, Title: pointers.New("Chat 2"), Members: members2, Messages: messages2},
			},
			wantErr: false,
		},
		{
			name: "error getting user chats",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
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
			name: "error getting chat members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(chats, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(nil, errors.New("members error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("members error"),
		},
		{
			name: "error getting chat messages",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(chats, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(members1, nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), messagesPagination).
						Return(nil, errors.New("messages error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("messages error"),
		},
		{
			name: "empty chat list",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(2), pagination).
						Return([]domains.Chat{}, nil)
					// No calls for members or messages since no chats
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
		{
			name: "chat with no messages",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					singleChat := []domains.Chat{{ID: 102, Title: pointers.New("Empty Chat")}}
					cr.EXPECT().
						GetUserChats(gomock.Any(), uint64(1), pagination).
						Return(singleChat, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(102)).
						Return(members1, nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(102), messagesPagination).
						Return([]domains.Message{}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				pagination: pagination,
			},
			want: []domains.Chat{
				{
					ID:       102,
					Title:    pointers.New("Empty Chat"),
					Members:  members1,
					Messages: []domains.Message{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockChatsRepo := mockrepositories.NewMockChatsRepository(ctrl)
			mockMessagesRepo := mockrepositories.NewMockMessagesRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockChatsRepository != nil {
				tt.fields.mockChatsRepository(mockChatsRepo)
			}

			if tt.fields.mockMessagesRepository != nil {
				tt.fields.mockMessagesRepository(mockMessagesRepo)
			}

			newChatsRepoFunc := func(_ pg.Transaction) interfaces.ChatsRepository {
				return mockChatsRepo
			}

			newMessagesRepoFunc := func(_ pg.Transaction) interfaces.MessagesRepository {
				return mockMessagesRepo
			}

			s := service.New(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := s.GetUserChats(tt.args.ctx, tt.args.userID, tt.args.pagination)

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

func TestService_CreateChat(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx  context.Context
		chat domains.Chat
	}

	chatToCreate := domains.Chat{
		Title: pointers.New("New Chat"),
		Type:  domains.ChatTypeGroup,
		Members: []domains.User{
			{ID: 1, Username: "user1"},
			{ID: 2, Username: "user2"},
		},
	}

	createdChatID := uint64(100)
	createdChat := &domains.Chat{
		ID:    createdChatID,
		Title: pointers.New("New Chat"),
		Type:  domains.ChatTypeGroup,
	}

	members := []domains.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Chat
		wantErr bool
		err     error
	}{
		{
			name: "successfully create chat",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						CreateChat(gomock.Any(), chatToCreate).
						Return(createdChatID, nil)

					cr.EXPECT().
						GetChatByID(gomock.Any(), createdChatID).
						Return(createdChat, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), createdChatID).
						Return(members, nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: chatToCreate,
			},
			want: &domains.Chat{
				ID:      createdChatID,
				Title:   pointers.New("New Chat"),
				Type:    domains.ChatTypeGroup,
				Members: members,
			},
			wantErr: false,
		},
		{
			name: "error creating chat",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						CreateChat(gomock.Any(), chatToCreate).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: chatToCreate,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting created chat",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						CreateChat(gomock.Any(), chatToCreate).
						Return(createdChatID, nil)

					cr.EXPECT().
						GetChatByID(gomock.Any(), createdChatID).
						Return(nil, errors.New("chat not found"))
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: chatToCreate,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("chat not found"),
		},
		{
			name: "error getting chat members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						CreateChat(gomock.Any(), chatToCreate).
						Return(createdChatID, nil)

					cr.EXPECT().
						GetChatByID(gomock.Any(), createdChatID).
						Return(createdChat, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), createdChatID).
						Return(nil, errors.New("members error"))
				},
			},
			args: args{
				ctx:  context.Background(),
				chat: chatToCreate,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("members error"),
		},
		{
			name: "create private chat",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					privateChat := domains.Chat{
						Type: domains.ChatTypePrivate,
						Members: []domains.User{
							{ID: 1, Username: "user1"},
							{ID: 2, Username: "user2"},
						},
					}

					cr.EXPECT().
						CreateChat(gomock.Any(), privateChat).
						Return(uint64(101), nil)

					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(101)).
						Return(&domains.Chat{
							ID:   101,
							Type: domains.ChatTypePrivate,
						}, nil)

					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(101)).
						Return(members, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Type: domains.ChatTypePrivate,
					Members: []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
					},
				},
			},
			want: &domains.Chat{
				ID:      101,
				Type:    domains.ChatTypePrivate,
				Members: members,
			},
			wantErr: false,
		},
		{
			name: "create chat with single member",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					singleMemberChat := domains.Chat{
						Title: pointers.New("Self Chat"),
						Type:  domains.ChatTypeGroup,
						Members: []domains.User{
							{ID: 1, Username: "user1"},
						},
					}

					cr.EXPECT().
						CreateChat(gomock.Any(), singleMemberChat).
						Return(uint64(102), nil)

					cr.EXPECT().
						GetChatByID(gomock.Any(), uint64(102)).
						Return(&domains.Chat{
							ID:    102,
							Title: pointers.New("Self Chat"),
							Type:  domains.ChatTypeGroup,
						}, nil)

					singleMember := []domains.User{{ID: 1, Username: "user1"}}
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(102)).
						Return(singleMember, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				chat: domains.Chat{
					Title: pointers.New("Self Chat"),
					Type:  domains.ChatTypeGroup,
					Members: []domains.User{
						{ID: 1, Username: "user1"},
					},
				},
			},
			want: &domains.Chat{
				ID:      102,
				Title:   pointers.New("Self Chat"),
				Type:    domains.ChatTypeGroup,
				Members: []domains.User{{ID: 1, Username: "user1"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockChatsRepo := mockrepositories.NewMockChatsRepository(ctrl)
			mockMessagesRepo := mockrepositories.NewMockMessagesRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockChatsRepository != nil {
				tt.fields.mockChatsRepository(mockChatsRepo)
			}

			if tt.fields.mockMessagesRepository != nil {
				tt.fields.mockMessagesRepository(mockMessagesRepo)
			}

			newChatsRepoFunc := func(_ pg.Transaction) interfaces.ChatsRepository {
				return mockChatsRepo
			}

			newMessagesRepoFunc := func(_ pg.Transaction) interfaces.MessagesRepository {
				return mockMessagesRepo
			}

			s := service.New(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := s.CreateChat(tt.args.ctx, tt.args.chat)

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

func TestService_PrivateChatExists(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx     context.Context
		members []domains.User
	}

	members := []domains.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
		err     error
	}{
		{
			name: "private chat exists",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), members).
						Return(true, nil)
				},
			},
			args: args{
				ctx:     context.Background(),
				members: members,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "private chat does not exist",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), members).
						Return(false, nil)
				},
			},
			args: args{
				ctx:     context.Background(),
				members: members,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "error checking private chat existence",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), members).
						Return(false, errors.New("database error"))
				},
			},
			args: args{
				ctx:     context.Background(),
				members: members,
			},
			want:    false,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "check private chat with single member",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					singleMember := []domains.User{{ID: 1, Username: "user1"}}
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), singleMember).
						Return(false, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				members: []domains.User{
					{ID: 1, Username: "user1"},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "check private chat with multiple members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					multipleMembers := []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
						{ID: 3, Username: "user3"},
					}
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), multipleMembers).
						Return(false, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				members: []domains.User{
					{ID: 1, Username: "user1"},
					{ID: 2, Username: "user2"},
					{ID: 3, Username: "user3"},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "empty members list",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						PrivateChatExists(gomock.Any(), []domains.User{}).
						Return(false, nil)
				},
			},
			args: args{
				ctx:     context.Background(),
				members: []domains.User{},
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockChatsRepo := mockrepositories.NewMockChatsRepository(ctrl)
			mockMessagesRepo := mockrepositories.NewMockMessagesRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockChatsRepository != nil {
				tt.fields.mockChatsRepository(mockChatsRepo)
			}

			if tt.fields.mockMessagesRepository != nil {
				tt.fields.mockMessagesRepository(mockMessagesRepo)
			}

			newChatsRepoFunc := func(_ pg.Transaction) interfaces.ChatsRepository {
				return mockChatsRepo
			}

			newMessagesRepoFunc := func(_ pg.Transaction) interfaces.MessagesRepository {
				return mockMessagesRepo
			}

			s := service.New(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := s.PrivateChatExists(tt.args.ctx, tt.args.members)

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
