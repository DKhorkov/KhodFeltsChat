package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	messagesTableName  = "messages"
	senderIDColumnName = "sender_id"
	textColumnName     = "text"
)

type MessagesRepository struct {
	tx pg.Transaction
}

func NewMessagesRepository(
	tx pg.Transaction,
) *MessagesRepository {
	return &MessagesRepository{
		tx: tx,
	}
}

func (repo *MessagesRepository) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(messagesTableName).
		Columns(
			chatIDColumnName,
			senderIDColumnName,
			textColumnName,
		).
		Values(
			message.ChatID,
			message.Sender.ID,
			message.Text,
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
	if err != nil {
		return 0, err
	}

	var messageID uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&messageID); err != nil {
		return 0, err
	}

	return messageID, nil
}

func (repo *MessagesRepository) GetChatMessages(
	ctx context.Context,
	chatID uint64,
	pagination *domains.Pagination,
) ([]domains.Message, error) {
	columnsForSelect := []string{
		fmt.Sprintf("%s.%s", messagesTableName, idColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName),
		fmt.Sprintf("%s.%s", usersTableName, idColumnName),
		fmt.Sprintf("%s.%s", usersTableName, usernameColumnName),
		fmt.Sprintf("%s.%s", usersTableName, emailColumnName),
		fmt.Sprintf("%s.%s", usersTableName, emailConfirmedColumnName),
		fmt.Sprintf("%s.%s", usersTableName, passwordColumnName),
		fmt.Sprintf("%s.%s", usersTableName, createdAtColumnName),
		fmt.Sprintf("%s.%s", usersTableName, updatedAtColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, textColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, createdAtColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, updatedAtColumnName),
	}

	builder := sq.
		Select(columnsForSelect...).
		From(messagesTableName).
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				usersTableName,
				usersTableName,
				idColumnName,
				messagesTableName,
				senderIDColumnName,
			),
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName): chatID,
			},
		).
		OrderBy(fmt.Sprintf("%s.%s %s", messagesTableName, idColumnName, desc)).
		PlaceholderFormat(sq.Dollar)

	if pagination != nil && pagination.Limit != nil {
		builder = builder.Limit(*pagination.Limit)
	}

	if pagination != nil && pagination.Offset != nil {
		builder = builder.Offset(*pagination.Offset)
	}

	stmt, params, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := repo.tx.QueryContext(
		ctx,
		stmt,
		params...,
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		rowsErr := rows.Close()
		if rowsErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; %w", err, rowsErr)

				return
			}

			err = rowsErr
		}
	}()

	var messages []domains.Message

	for rows.Next() {
		messagePg := MessagePg{}
		columns := pg.GetEntityColumns(&messagePg) // Only pointer to use rows.Scan() successfully

		err = rows.Scan(columns...)
		if err != nil {
			return nil, err
		}

		messages = append(messages, *repo.pgMessageToDomainMessage(messagePg))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (repo *MessagesRepository) GetMessageByID(
	ctx context.Context,
	id uint64,
) (*domains.Message, error) {
	columnsForSelect := []string{
		fmt.Sprintf("%s.%s", messagesTableName, idColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName),
		fmt.Sprintf("%s.%s", usersTableName, idColumnName),
		fmt.Sprintf("%s.%s", usersTableName, usernameColumnName),
		fmt.Sprintf("%s.%s", usersTableName, emailColumnName),
		fmt.Sprintf("%s.%s", usersTableName, emailConfirmedColumnName),
		fmt.Sprintf("%s.%s", usersTableName, passwordColumnName),
		fmt.Sprintf("%s.%s", usersTableName, createdAtColumnName),
		fmt.Sprintf("%s.%s", usersTableName, updatedAtColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, textColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, createdAtColumnName),
		fmt.Sprintf("%s.%s", messagesTableName, updatedAtColumnName),
	}

	stmt, params, err := sq.
		Select(columnsForSelect...).
		From(messagesTableName).
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				usersTableName,
				usersTableName,
				idColumnName,
				messagesTableName,
				senderIDColumnName,
			),
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesTableName, idColumnName): id,
			},
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	messagePg := &MessagePg{}

	columns := pg.GetEntityColumns(messagePg)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return repo.pgMessageToDomainMessage(*messagePg), nil
}

func (repo *MessagesRepository) pgMessageToDomainMessage(messagePg MessagePg) *domains.Message {
	return &domains.Message{
		ID:     messagePg.ID,
		ChatID: messagePg.ChatID,
		Sender: domains.User{
			ID:             messagePg.SenderID,
			Username:       messagePg.SenderUsername,
			Email:          messagePg.SenderEmail,
			EmailConfirmed: messagePg.SenderEmailConfirmed,
			Password:       messagePg.SenderPassword,
			CreatedAt:      messagePg.CreatedAt,
			UpdatedAt:      messagePg.UpdatedAt,
		},
		Text:      messagePg.Text,
		CreatedAt: messagePg.CreatedAt,
		UpdatedAt: messagePg.UpdatedAt,
	}
}

type MessagePg struct {
	ID                   uint64
	ChatID               uint64
	SenderID             uint64
	SenderUsername       string
	SenderEmail          string
	SenderEmailConfirmed bool
	SenderPassword       string
	SenderCreatedAt      time.Time
	SenderUpdatedAt      time.Time
	Text                 string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
