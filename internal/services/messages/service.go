package messages

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
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

func (s *Service) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (*domains.Message, error) {
	var createdMessage *domains.Message

	err := s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)

			messageID, err := messagesRepository.SaveMessage(ctx, message)
			if err != nil {
				return err
			}

			if createdMessage, err = messagesRepository.GetMessageByID(
				ctx,
				message.Sender.ID,
				messageID,
			); err != nil {
				return err
			}

			if err = messagesRepository.ReadAllChatMessages(
				ctx,
				message.Sender.ID,
				message.ChatID,
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return createdMessage, nil
}

func (s *Service) GetChatMessages(
	ctx context.Context,
	userID uint64,
	chatID uint64,
	pagination *domains.Pagination,
) ([]domains.Message, error) {
	var (
		messages []domains.Message
		err      error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			chatsRepository := s.newChatsRepositoryFunc(tx)
			if _, err = chatsRepository.GetChatByID(ctx, chatID); err != nil {
				return fmt.Errorf("%w: %w", customerrors.ErrChatNotFound, err)
			}

			messagesRepository := s.newMessagesRepositoryFunc(tx)
			if messages, err = messagesRepository.GetChatMessages(
				ctx,
				userID,
				chatID,
				pagination,
			); err != nil {
				return err
			}

			if err = messagesRepository.ChangeMessagesIsReadStatus(
				ctx,
				userID,
				messages,
				true, // При получении сообщений помечаем их прочитанными для текущего пользователя
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *Service) GetMessageByID(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) (*domains.Message, error) {
	var (
		message *domains.Message
		err     error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)
			message, err = messagesRepository.GetMessageByID(ctx, userID, messageID)

			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *Service) GetUserUnreadCount(
	ctx context.Context,
	userID uint64,
) (uint64, error) {
	var count uint64

	err := s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)

			var err error

			count, err = messagesRepository.GetUserUnreadCount(ctx, userID)

			return err
		},
	)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) DeleteMessage(
	ctx context.Context,
	dto domains.DeleteMessageDTO,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)

			if dto.ForAll {
				return messagesRepository.DeleteMessageForAll(ctx, dto.MessageID)
			}

			return messagesRepository.DeleteMessageForUser(ctx, dto.UserID, dto.MessageID)
		},
	)
}

func (s *Service) UpdateMessage(
	ctx context.Context,
	dto domains.UpdateMessageDTO,
) (*domains.Message, error) {
	var updatedMessage *domains.Message

	err := s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			messagesRepository := s.newMessagesRepositoryFunc(tx)

			if err := messagesRepository.UpdateMessage(ctx, dto); err != nil {
				return err
			}

			var err error
			if updatedMessage, err = messagesRepository.GetMessageByID(
				ctx,
				dto.UserID,
				dto.MessageID,
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return updatedMessage, nil
}
