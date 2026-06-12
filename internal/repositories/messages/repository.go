package messages

import (
	"context"
	"fmt"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/logging"
	sq "github.com/Masterminds/squirrel"
)

const (
	messagesTableName         = "messages"
	messagesStatusesTableName = "messages_statuses"
	usersTableName            = "users"
	chatsMembersTableName     = "chats_members"

	senderIDColumnName         = "sender_id"
	textColumnName             = "text"
	messageIDColumnName        = "message_id"
	idColumnName               = "id"
	usernameColumnName         = "username"
	emailColumnName            = "email"
	createdAtColumnName        = "created_at"
	updatedAtColumnName        = "updated_at"
	userIDColumnName           = "user_id"
	emailConfirmedColumnName   = "email_confirmed"
	passwordColumnName         = "password"
	chatIDColumnName           = "chat_id"
	isReadColumnName           = "is_read"
	replyToMessageIDColumnName = "reply_to_message_id"
	isDeletedColumnName        = "is_deleted"

	replyTableAlias       = "reply"
	replySenderTableAlias = "reply_sender"

	desc = "DESC"
	asc  = "ASC"

	returningIDSuffix = "RETURNING id"
)

type Repository struct {
	tx     pg.Transaction
	logger logging.Logger
}

func New(
	tx pg.Transaction,
	logger logging.Logger,
) *Repository {
	return &Repository{
		tx:     tx,
		logger: logger,
	}
}

func (repo *Repository) SaveMessage(
	ctx context.Context,
	message domains.Message,
) (uint64, error) {
	var replyToMessageID *uint64
	if message.ReplyToMessage != nil {
		replyToMessageID = &message.ReplyToMessage.ID
	}

	stmt, params, err := sq.
		Insert(messagesTableName).
		Columns(
			chatIDColumnName,
			senderIDColumnName,
			textColumnName,
			replyToMessageIDColumnName,
		).
		Values(
			message.ChatID,
			message.Sender.ID,
			message.Text,
			replyToMessageID,
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

	// Достаем id всех участников чата
	stmt, params, err = sq.
		Select(userIDColumnName).
		From(chatsMembersTableName).
		Where(sq.Eq{chatIDColumnName: message.ChatID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, err
	}

	rows, err := repo.tx.QueryContext(
		ctx,
		stmt,
		params...,
	)
	if err != nil {
		return 0, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, repo.logger, "Failed to close SQL rows", err)
		}
	}()

	var userIDs []uint64

	for rows.Next() {
		var userID uint64

		err = rows.Scan(&userID)
		if err != nil {
			return 0, err
		}

		userIDs = append(userIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return 0, err
	}

	// Сохраняем статус нового сообщения для всех участников чата
	builder := sq.
		Insert(messagesStatusesTableName).
		Columns(
			messageIDColumnName,
			userIDColumnName,
			isReadColumnName,
		).
		PlaceholderFormat(sq.Dollar)

	for _, userID := range userIDs {
		isRead := userID == message.Sender.ID // true для отправителя, false для остальных
		builder = builder.Values(messageID, userID, isRead)
	}

	stmt, params, err = builder.ToSql()
	if err != nil {
		return 0, err
	}

	if _, err = repo.tx.ExecContext(
		ctx,
		stmt,
		params...,
	); err != nil {
		return 0, err
	}

	return messageID, nil
}

func (repo *Repository) GetChatMessages(
	ctx context.Context,
	userID uint64,
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
		fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName),
		// Reply fields:
		fmt.Sprintf("%s.%s", replyTableAlias, idColumnName),
		fmt.Sprintf("%s.%s", replyTableAlias, textColumnName),
		fmt.Sprintf("%s.%s", replyTableAlias, createdAtColumnName),
		fmt.Sprintf("%s.%s", replySenderTableAlias, idColumnName),
		fmt.Sprintf("%s.%s", replySenderTableAlias, usernameColumnName),
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
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				messagesStatusesTableName,
				messagesStatusesTableName,
				messageIDColumnName,
				messagesTableName,
				idColumnName,
			),
		).
		LeftJoin(
			fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				messagesTableName, replyTableAlias,
				messagesTableName, replyToMessageIDColumnName,
				replyTableAlias, idColumnName,
			),
		).
		LeftJoin(
			fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				usersTableName, replySenderTableAlias,
				replyTableAlias, senderIDColumnName,
				replySenderTableAlias, idColumnName,
			),
		).
		Where(
			sq.And{
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesTableName, chatIDColumnName): chatID,
				},
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
				},
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, isDeletedColumnName): false,
				},
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
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, repo.logger, "Failed to close SQL rows", err)
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

		messages = append(messages, *pgMessageToDomainMessage(messagePg))
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (repo *Repository) GetMessageByID(
	ctx context.Context,
	userID uint64,
	messageID uint64,
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
		fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName),
		// Reply fields:
		fmt.Sprintf("%s.%s", replyTableAlias, idColumnName),
		fmt.Sprintf("%s.%s", replyTableAlias, textColumnName),
		fmt.Sprintf("%s.%s", replyTableAlias, createdAtColumnName),
		fmt.Sprintf("%s.%s", replySenderTableAlias, idColumnName),
		fmt.Sprintf("%s.%s", replySenderTableAlias, usernameColumnName),
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
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				messagesStatusesTableName,
				messagesStatusesTableName,
				messageIDColumnName,
				messagesTableName,
				idColumnName,
			),
		).
		LeftJoin(
			fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				messagesTableName, replyTableAlias,
				messagesTableName, replyToMessageIDColumnName,
				replyTableAlias, idColumnName,
			),
		).
		LeftJoin(
			fmt.Sprintf(
				"%s AS %s ON %s.%s = %s.%s",
				usersTableName, replySenderTableAlias,
				replyTableAlias, senderIDColumnName,
				replySenderTableAlias, idColumnName,
			),
		).
		Where(
			sq.And{
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesTableName, idColumnName): messageID,
				},
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
				},
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, isDeletedColumnName): false,
				},
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

	return pgMessageToDomainMessage(*messagePg), nil
}

func (repo *Repository) ChangeMessagesIsReadStatus(
	ctx context.Context,
	userID uint64,
	messages []domains.Message,
	isRead bool,
) error {
	messageIDs := make([]uint64, 0, len(messages))
	for i := range messages {
		messageIDs = append(messageIDs, messages[i].ID)
	}

	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Where(
			sq.Eq{
				userIDColumnName:    userID,
				messageIDColumnName: messageIDs,
			},
		).
		Set(isReadColumnName, isRead).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) ReadAllChatMessages(
	ctx context.Context,
	userID uint64,
	chatID uint64,
) error {
	chatMessagesSubquery := sq.
		Select(idColumnName).
		From(messagesTableName).
		Where(sq.Eq{chatIDColumnName: chatID})

	chatMessagesSQL, chatMessagesArgs, err := chatMessagesSubquery.ToSql()
	if err != nil {
		return err
	}

	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Set(isReadColumnName, true).
		Where(
			sq.And{
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
				},
				sq.Eq{
					fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName): false,
				},
				sq.Expr(
					fmt.Sprintf("%s IN (%s)", messageIDColumnName, chatMessagesSQL),
					chatMessagesArgs...,
				),
			},
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) GetUserUnreadCount(
	ctx context.Context,
	userID uint64,
) (uint64, error) {
	stmt, params, err := sq.
		Select("COUNT(*)").
		From(messagesStatusesTableName).
		Where(sq.Eq{userIDColumnName: userID}).
		Where(sq.Eq{isReadColumnName: false}).
		Where(sq.Eq{isDeletedColumnName: false}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, err
	}

	var count uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (repo *Repository) DeleteMessageForUser(
	ctx context.Context,
	userID uint64,
	messageID uint64,
) error {
	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Set(isDeletedColumnName, true).
		Where(sq.Eq{
			userIDColumnName:    userID,
			messageIDColumnName: messageID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) DeleteMessageForAll(
	ctx context.Context,
	messageID uint64,
) error {
	stmt, params, err := sq.
		Update(messagesStatusesTableName).
		Set(isDeletedColumnName, true).
		Where(sq.Eq{
			messageIDColumnName: messageID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) UpdateMessage(
	ctx context.Context,
	dto domains.UpdateMessageDTO,
) error {
	stmt, params, err := sq.
		Update(messagesTableName).
		Set(textColumnName, dto.Text).
		Set(updatedAtColumnName, time.Now()).
		Where(sq.Eq{
			idColumnName: dto.MessageID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func pgMessageToDomainMessage(messagePg MessagePg) *domains.Message {
	msg := &domains.Message{
		ID:     messagePg.ID,
		ChatID: messagePg.ChatID,
		Sender: domains.User{
			ID:             messagePg.SenderID,
			Username:       messagePg.SenderUsername,
			Email:          messagePg.SenderEmail,
			EmailConfirmed: messagePg.SenderEmailConfirmed,
			Password:       messagePg.SenderPassword,
			CreatedAt:      messagePg.SenderCreatedAt,
			UpdatedAt:      messagePg.SenderUpdatedAt,
		},
		Text:      messagePg.Text,
		CreatedAt: messagePg.CreatedAt,
		UpdatedAt: messagePg.UpdatedAt,
		IsRead:    messagePg.IsRead,
	}

	if messagePg.ReplyToMessageID != nil {
		msg.ReplyToMessage = &domains.Message{
			ID:   *messagePg.ReplyToMessageID,
			Text: *messagePg.ReplyToMessageText,
			Sender: domains.User{
				ID:       *messagePg.ReplyToSenderID,
				Username: *messagePg.ReplyToSenderUsername,
			},
			CreatedAt: *messagePg.ReplyToMessageCreatedAt,
		}
	}

	return msg
}

type MessagePg struct {
	ID                      uint64
	ChatID                  uint64
	SenderID                uint64
	SenderUsername          string
	SenderEmail             string
	SenderEmailConfirmed    bool
	SenderPassword          string
	SenderCreatedAt         time.Time
	SenderUpdatedAt         time.Time
	Text                    string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	IsRead                  bool
	ReplyToMessageID        *uint64
	ReplyToMessageText      *string
	ReplyToMessageCreatedAt *time.Time
	ReplyToSenderID         *uint64
	ReplyToSenderUsername   *string
}
