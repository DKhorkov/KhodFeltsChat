package services

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/pointers"
)

type ChatsService struct {
	uow                       interfaces.UnitOfWork
	newChatsRepositoryFunc    func(tx pg.Transaction) interfaces.ChatsRepository
	newMessagesRepositoryFunc func(tx pg.Transaction) interfaces.MessagesRepository
}

func NewChatsService(
	uow interfaces.UnitOfWork,
	newChatsRepositoryFunc func(tx pg.Transaction) interfaces.ChatsRepository,
	newMessagesRepositoryFunc func(tx pg.Transaction) interfaces.MessagesRepository,
) *ChatsService {
	return &ChatsService{
		uow:                       uow,
		newChatsRepositoryFunc:    newChatsRepositoryFunc,
		newMessagesRepositoryFunc: newMessagesRepositoryFunc,
	}
}

func (s *ChatsService) GetChatMembers(
	ctx context.Context,
	chatID uint64,
) ([]domains.User, error) {
	var (
		members []domains.User
		err     error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)
			if _, err = chatsRepository.GetChatByID(ctx, chatID); err != nil {
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

func (s *ChatsService) GetUserChats(
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
				var members []domains.User
				if members, err = chatsRepository.GetChatMembers(ctx, chats[i].ID); err != nil {
					return err
				}

				chats[i].Members = members

				messages, err := messageRepository.GetChatMessages(
					ctx,
					chats[i].ID,
					messagesPagination,
				)
				if err != nil {
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

func (s *ChatsService) CreateChat(
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

			if createdChat, err = chatsRepository.GetChatByID(ctx, chatID); err != nil {
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
