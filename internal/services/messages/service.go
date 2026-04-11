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

			chatsRepository := s.newChatsRepositoryFunc(tx)

			members, err := chatsRepository.GetChatMembers(ctx, message.ChatID)
			if err != nil {
				return err
			}

			for _, member := range members {
				// Не меняем статус просмотра для того, кто отправил сообщение:
				if member.ID == message.Sender.ID {
					continue
				}

				if err = chatsRepository.ChangeChatIsReadStatus(
					ctx,
					member.ID,
					message.ChatID,
					false, // Чат не просмотрен, поскольку было получено новое сообщение
				); err != nil {
					return err
				}
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

			if err = chatsRepository.ChangeChatIsReadStatus(
				ctx,
				userID,
				chatID,
				true, // При получении сообщений чат считается просмотренным для текущего пользователя
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
