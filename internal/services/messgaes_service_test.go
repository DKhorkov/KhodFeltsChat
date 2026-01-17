package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/services"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestMessagesService_SaveMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx     context.Context
		message domains.Message
	}

	messageToSave := domains.Message{
		ChatID: 100,
		Sender: domains.User{ID: 1, Username: "user1"},
		Text:   "Hello, world!",
	}

	savedMessageID := uint64(500)
	savedMessage := &domains.Message{
		ID:     savedMessageID,
		ChatID: 100,
		Sender: domains.User{ID: 1, Username: "user1"},
		Text:   "Hello, world!",
	}

	chatMembers := []domains.User{
		{ID: 1, Username: "user1"},
		{ID: 2, Username: "user2"},
		{ID: 3, Username: "user3"},
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
			name: "successfully save message with multiple recipients",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(chatMembers, nil)

					// Для user2 и user3 меняем статус на непрочитанный
					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(2), uint64(100), false).
						Return(nil)

					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(3), uint64(100), false).
						Return(nil)
					// Для user1 (отправитель) не должно быть вызова
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    savedMessage,
			wantErr: false,
		},
		{
			name: "successfully save message with single recipient (sender only)",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(domains.Message{})).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					singleMember := []domains.User{{ID: 1, Username: "user1"}}
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(singleMember, nil)
					// Нет вызовов ChangeChatIsReadStatus, так как отправитель - единственный участник
				},
			},
			args: args{
				ctx: context.Background(),
				message: domains.Message{
					ChatID: 100,
					Sender: domains.User{ID: 1},
					Text:   "Hello!",
				},
			},
			want:    savedMessage,
			wantErr: false,
		},
		{
			name: "error saving message",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(uint64(0), errors.New("database error"))
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting saved message",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(nil, errors.New("message not found"))
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("message not found"),
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
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(nil, errors.New("members error"))
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("members error"),
		},
		{
			name: "error changing chat read status",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return(chatMembers, nil)

					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(2), uint64(100), false).
						Return(errors.New("status update error"))
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("status update error"),
		},
		{
			name: "empty chat members list",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), messageToSave).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(100)).
						Return([]domains.User{}, nil)
					// Нет вызовов ChangeChatIsReadStatus, так как нет участников
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    savedMessage,
			wantErr: false,
		},
		{
			name: "save message in group chat with many members",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						SaveMessage(gomock.Any(), gomock.AssignableToTypeOf(domains.Message{})).
						Return(savedMessageID, nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), savedMessageID).
						Return(savedMessage, nil)
				},
				mockChatsRepository: func(cr *mockrepositories.MockChatsRepository) {
					groupMembers := []domains.User{
						{ID: 1, Username: "user1"},
						{ID: 2, Username: "user2"},
						{ID: 3, Username: "user3"},
						{ID: 4, Username: "user4"},
						{ID: 5, Username: "user5"},
					}
					cr.EXPECT().
						GetChatMembers(gomock.Any(), uint64(200)).
						Return(groupMembers, nil)

					// Для всех кроме отправителя (user1) меняем статус
					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(2), uint64(200), false).
						Return(nil)

					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(3), uint64(200), false).
						Return(nil)

					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(4), uint64(200), false).
						Return(nil)

					cr.EXPECT().
						ChangeChatIsReadStatus(gomock.Any(), uint64(5), uint64(200), false).
						Return(nil)
				},
			},
			args: args{
				ctx: context.Background(),
				message: domains.Message{
					ChatID: 200,
					Sender: domains.User{ID: 1},
					Text:   "Group message",
				},
			},
			want:    savedMessage,
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

			service := services.NewMessagesService(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := service.SaveMessage(tt.args.ctx, tt.args.message)

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

func TestMessagesService_GetChatMessages(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx        context.Context
		userID     uint64
		chatID     uint64
		pagination *domains.Pagination
	}

	chat := &domains.Chat{ID: 100, Title: pointers.New("Test Chat")}
	messages := []domains.Message{
		{ID: 1, ChatID: 100, Text: "Message 1", Sender: domains.User{ID: 1}},
		{ID: 2, ChatID: 100, Text: "Message 2", Sender: domains.User{ID: 2}},
	}

	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](10),
		Offset: pointers.New[uint64](0),
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(1), uint64(100), true).
						Return(nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), pagination).
						Return(messages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: pagination,
			},
			want:    messages,
			wantErr: false,
		},
		{
			name: "successfully get chat messages without pagination",
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(1), uint64(100), true).
						Return(nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), (*domains.Pagination)(nil)).
						Return(messages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: nil,
			},
			want:    messages,
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
				ctx:        context.Background(),
				userID:     1,
				chatID:     999,
				pagination: pagination,
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
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrChatNotFound,
		},
		{
			name: "error getting messages",
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
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), pagination).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error changing chat read status",
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(1), uint64(100), true).
						Return(errors.New("status update error"))
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), pagination).
						Return(messages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     100,
				pagination: pagination,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("status update error"),
		},
		{
			name: "empty messages list",
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(1), uint64(101), true).
						Return(nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(101), pagination).
						Return([]domains.Message{}, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     1,
				chatID:     101,
				pagination: pagination,
			},
			want:    []domains.Message{},
			wantErr: false,
		},
		{
			name: "get messages with custom pagination values",
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(1), uint64(100), true).
						Return(nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					customPagination := &domains.Pagination{
						Limit:  pointers.New[uint64](50),
						Offset: pointers.New[uint64](20),
					}
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), customPagination).
						Return(messages, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 1,
				chatID: 100,
				pagination: &domains.Pagination{
					Limit:  pointers.New[uint64](50),
					Offset: pointers.New[uint64](20),
				},
			},
			want:    messages,
			wantErr: false,
		},
		{
			name: "get messages for different user",
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
						ChangeChatIsReadStatus(gomock.Any(), uint64(2), uint64(100), true).
						Return(nil)
				},
				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(100), pagination).
						Return(messages, nil)
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     2,
				chatID:     100,
				pagination: pagination,
			},
			want:    messages,
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

			service := services.NewMessagesService(
				mockUOW,
				newChatsRepoFunc,
				newMessagesRepoFunc,
			)

			// Act
			got, err := service.GetChatMessages(
				tt.args.ctx,
				tt.args.userID,
				tt.args.chatID,
				tt.args.pagination,
			)

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
