package chats

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
)

type Service struct {
	uow                       interfaces.UnitOfWork
	newChatsRepositoryFunc    func(tx pg.Transaction) interfaces.ChatsRepository
	newMessagesRepositoryFunc func(tx pg.Transaction) interfaces.MessagesRepository
}

func New(
	uow interfaces.UnitOfWork,
	newChatsRepositoryFunc func(tx pg.Transaction) interfaces.ChatsRepository,
	newMessagesRepositoryFunc func(tx pg.Transaction) interfaces.MessagesRepository,
) *Service {
	return &Service{
		uow:                       uow,
		newChatsRepositoryFunc:    newChatsRepositoryFunc,
		newMessagesRepositoryFunc: newMessagesRepositoryFunc,
	}
}

func (s *Service) GetChatByID(
	ctx context.Context,
	chatID, userID uint64,
) (*domains.Chat, error) {
	var (
		chat *domains.Chat
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)
			if chat, err = chatsRepository.GetChatByID(ctx, chatID, userID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrChatNotFound, err)
	}

	return chat, nil
}

func (s *Service) GetChatMembers(
	ctx context.Context,
	chatID, userID uint64,
) ([]domains.User, error) {
	var (
		members []domains.User
		err     error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)
			if _, err = chatsRepository.GetChatByID(ctx, chatID, userID); err != nil {
				return fmt.Errorf("%w: %w", customerrors.ErrChatNotFound, err)
			}

			if members, err = chatsRepository.GetChatMembers(ctx, chatID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return members, nil
}

func (s *Service) GetUserChats(
	ctx context.Context,
	userID uint64,
	pagination *domains.Pagination,
) ([]domains.Chat, error) {
	var (
		chats []domains.Chat
		err   error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)
			if chats, err = chatsRepository.GetUserChats(ctx, userID, pagination); err != nil {
				return err
			}

			messageRepository := s.newMessagesRepositoryFunc(tx)
			messagesPagination := &domains.Pagination{
				Limit: pointers.New[uint64](1), // Только последнее сообщение
			}

			for i := range chats {
				var (
					members  []domains.User
					messages []domains.Message
				)

				if members, err = chatsRepository.GetChatMembers(ctx, chats[i].ID); err != nil {
					return err
				}

				chats[i].Members = members

				if messages, err = messageRepository.GetChatMessages(
					ctx,
					userID,
					chats[i].ID,
					messagesPagination,
				); err != nil {
					return err
				}

				chats[i].Messages = messages
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *Service) CreateChat(
	ctx context.Context,
	chat domains.Chat,
) (*domains.Chat, error) {
	var createdChat *domains.Chat

	err := s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)

			chatID, err := chatsRepository.CreateChat(ctx, chat)
			if err != nil {
				return err
			}

			// userID=0 — у только что созданного чата непрочитанных всё равно ноль.
			if createdChat, err = chatsRepository.GetChatByID(ctx, chatID, 0); err != nil {
				return err
			}

			members, err := chatsRepository.GetChatMembers(ctx, chatID)
			if err != nil {
				return err
			}

			createdChat.Members = members

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return createdChat, nil
}

func (s *Service) PrivateChatExists(
	ctx context.Context,
	members []domains.User,
) (bool, error) {
	var (
		exists bool
		err    error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)

			if exists, err = chatsRepository.PrivateChatExists(ctx, members); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return false, err
	}

	return exists, nil
}
