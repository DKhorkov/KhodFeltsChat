package messages_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/messages"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_SaveMessage(t *testing.T) {
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
						GetMessageByID(gomock.Any(), uint64(1), savedMessageID).
						Return(savedMessage, nil)

					mr.EXPECT().
						ReadAllChatMessages(gomock.Any(), uint64(1), uint64(100)).
						Return(nil)
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
			name: "error reading all chat messages",
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
						GetMessageByID(gomock.Any(), uint64(1), savedMessageID).
						Return(savedMessage, nil)

					mr.EXPECT().
						ReadAllChatMessages(gomock.Any(), uint64(1), uint64(100)).
						Return(errors.New("read all error"))
				},
			},
			args: args{
				ctx:     context.Background(),
				message: messageToSave,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("read all error"),
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
						GetMessageByID(gomock.Any(), uint64(1), savedMessageID).
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
			got, err := s.SaveMessage(tt.args.ctx, tt.args.message)

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

func TestService_GetChatMessages(t *testing.T) {
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
				},

				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), pagination).
						Return(messages, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), messages, true).
						Return(nil)
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
				},

				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), (*domains.Pagination)(nil)).
						Return(messages, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), messages, true).
						Return(nil)
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
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), pagination).
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
				},

				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(101), pagination).
						Return([]domains.Message{}, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), []domains.Message{}, true).
						Return(nil)
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
				},

				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					customPagination := &domains.Pagination{
						Limit:  pointers.New[uint64](50),
						Offset: pointers.New[uint64](20),
					}

					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(1), uint64(100), customPagination).
						Return(messages, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(1), messages, true).
						Return(nil)
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
				},

				mockMessagesRepository: func(mr *mockrepositories.MockMessagesRepository) {
					mr.EXPECT().
						GetChatMessages(gomock.Any(), uint64(2), uint64(100), pagination).
						Return(messages, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(2), messages, true).
						Return(nil)
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
		{
			name: "ChangeMessagesIsReadStatus error",
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
						GetChatMessages(gomock.Any(), uint64(2), uint64(100), pagination).
						Return(messages, nil)

					mr.EXPECT().
						ChangeMessagesIsReadStatus(gomock.Any(), uint64(2), messages, true).
						Return(errors.New("test"))
				},
			},
			args: args{
				ctx:        context.Background(),
				userID:     2,
				chatID:     100,
				pagination: pagination,
			},
			wantErr: true,
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
			got, err := s.GetChatMessages(
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

func TestService_GetMessageByID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx       context.Context
		userID    uint64
		messageID uint64
	}

	expectedMessage := &domains.Message{
		ID:     1,
		ChatID: 100,
		Text:   "Hello, world!",
		Sender: domains.User{ID: 1, Username: "user1"},
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
			name: "successfully get message by ID",
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
						GetMessageByID(gomock.Any(), uint64(1), uint64(1)).
						Return(expectedMessage, nil)
				},
			},
			args: args{
				ctx:       context.Background(),
				userID:    1,
				messageID: 1,
			},
			want:    expectedMessage,
			wantErr: false,
		},
		{
			name: "repository error",
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
		{
			name: "UoW error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						Return(errors.New("unit of work error"))
				},
			},
			args: args{
				ctx:       context.Background(),
				userID:    1,
				messageID: 1,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("unit of work error"),
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
			got, err := s.GetMessageByID(tt.args.ctx, tt.args.userID, tt.args.messageID)

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

func TestService_DeleteMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
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
			name: "successfully delete for user",
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
						DeleteMessageForUser(gomock.Any(), uint64(1), uint64(10)).
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
			name: "successfully delete for all",
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
						DeleteMessageForAll(gomock.Any(), uint64(10)).
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
			name: "DeleteMessageForUser error",
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
						DeleteMessageForUser(gomock.Any(), uint64(1), uint64(10)).
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
		{
			name: "DeleteMessageForAll error",
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
						DeleteMessageForAll(gomock.Any(), uint64(10)).
						Return(errors.New("database error"))
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
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "UoW error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						Return(errors.New("unit of work error"))
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
			err:     errors.New("unit of work error"),
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
			err := s.DeleteMessage(tt.args.ctx, tt.args.dto)

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

func TestService_UpdateMessage(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockChatsRepository    func(*mockrepositories.MockChatsRepository)
		mockMessagesRepository func(*mockrepositories.MockMessagesRepository)
	}

	type args struct {
		ctx context.Context
		dto domains.UpdateMessageDTO
	}

	updateDTO := domains.UpdateMessageDTO{
		MessageID: 10,
		UserID:    1,
		Text:      "Updated text",
	}

	updatedMessage := &domains.Message{
		ID:     10,
		ChatID: 100,
		Text:   "Updated text",
		Sender: domains.User{ID: 1, Username: "user1"},
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
						UpdateMessage(gomock.Any(), updateDTO).
						Return(nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(updatedMessage, nil)
				},
			},
			args: args{
				ctx: context.Background(),
				dto: updateDTO,
			},
			want:    updatedMessage,
			wantErr: false,
		},
		{
			name: "UpdateMessage repository error",
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
						UpdateMessage(gomock.Any(), updateDTO).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: updateDTO,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "GetMessageByID repository error",
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
						UpdateMessage(gomock.Any(), updateDTO).
						Return(nil)

					mr.EXPECT().
						GetMessageByID(gomock.Any(), uint64(1), uint64(10)).
						Return(nil, errors.New("message not found"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: updateDTO,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("message not found"),
		},
		{
			name: "UoW error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						Return(errors.New("unit of work error"))
				},
			},
			args: args{
				ctx: context.Background(),
				dto: updateDTO,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("unit of work error"),
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
			got, err := s.UpdateMessage(tt.args.ctx, tt.args.dto)

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
