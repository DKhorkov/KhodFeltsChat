package chats

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/logging"
	sq "github.com/Masterminds/squirrel"
)

const (
	chatsTableName            = "chats"
	chatsMembersTableName     = "chats_members"
	usersTableName            = "users"
	messagesTableName         = "messages"
	messagesStatusesTableName = "messages_statuses"

	titleColumnName          = "title"
	descriptionColumnName    = "description"
	typeColumnName           = "type"
	chatIDColumnName         = "chat_id"
	isReadColumnName         = "is_read"
	isDeletedColumnName      = "is_deleted"
	messageIDColumnName      = "message_id"
	idColumnName             = "id"
	usernameColumnName       = "username"
	emailColumnName          = "email"
	createdAtColumnName      = "created_at"
	updatedAtColumnName      = "updated_at"
	userIDColumnName         = "user_id"
	emailConfirmedColumnName = "email_confirmed"
	passwordColumnName       = "password"
	avatarPathColumnName     = "avatar_path"

	existsSubquery = "EXISTS (?)"

	selectAllColumns = "*"
	selectExists     = "1"

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

func (repo *Repository) GetChatMembers(
	ctx context.Context,
	chatID uint64,
) ([]domains.User, error) {
	columnsForSelect := []string{
		idColumnName,
		usernameColumnName,
		emailColumnName,
		emailConfirmedColumnName,
		passwordColumnName,
		createdAtColumnName,
		updatedAtColumnName,
		avatarPathColumnName,
	}

	for i, column := range columnsForSelect {
		columnsForSelect[i] = fmt.Sprintf("%s.%s", usersTableName, column)
	}

	stmt, params, err := sq.
		Select(columnsForSelect...).
		From(usersTableName).
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				chatsMembersTableName,
				usersTableName,
				idColumnName,
				chatsMembersTableName,
				userIDColumnName,
			),
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", chatsMembersTableName, chatIDColumnName): chatID,
			},
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
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

	var users []domains.User

	for rows.Next() {
		user := domains.User{}
		columns := pg.GetEntityColumns(&user) // Only pointer to use rows.Scan() successfully

		err = rows.Scan(columns...)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (repo *Repository) GetUserChats(
	ctx context.Context,
	userID uint64,
	pagination *domains.Pagination,
) ([]domains.Chat, error) {
	unreadSubquery := sq.
		Select(selectExists).
		From(messagesStatusesTableName).
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				messagesTableName,
				messagesStatusesTableName,
				messageIDColumnName,
				messagesTableName,
				idColumnName,
			),
		).
		Where(
			sq.Expr(
				fmt.Sprintf(
					"%s.%s = %s.%s",
					messagesTableName,
					chatIDColumnName,
					chatsTableName,
					idColumnName,
				),
			),
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesStatusesTableName, userIDColumnName): userID,
			},
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesStatusesTableName, isReadColumnName): false,
			},
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", messagesStatusesTableName, isDeletedColumnName): false,
			},
		)

	unreadSQL, unreadArgs, err := unreadSubquery.ToSql()
	if err != nil {
		return nil, err
	}

	isReadColumn := fmt.Sprintf("NOT EXISTS (%s) AS %s", unreadSQL, isReadColumnName)

	columnsForSelect := []string{
		fmt.Sprintf("%s.%s", chatsTableName, idColumnName),
		fmt.Sprintf("%s.%s", chatsTableName, titleColumnName),
		fmt.Sprintf("%s.%s", chatsTableName, descriptionColumnName),
		fmt.Sprintf("%s.%s", chatsTableName, typeColumnName),
		fmt.Sprintf("%s.%s", chatsTableName, createdAtColumnName),
		fmt.Sprintf("%s.%s", chatsTableName, updatedAtColumnName),
	}

	builder := sq.
		Select(columnsForSelect...).
		Column(isReadColumn, unreadArgs...).
		From(chatsTableName).
		Join(
			fmt.Sprintf(
				"%s ON %s.%s = %s.%s",
				chatsMembersTableName,
				chatsTableName,
				idColumnName,
				chatsMembersTableName,
				chatIDColumnName,
			),
		).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", chatsMembersTableName, userIDColumnName): userID,
			},
		).
		OrderByClause(
			fmt.Sprintf(
				"COALESCE((SELECT MAX(%s) FROM %s WHERE %s = %s.%s), %s.%s) %s",
				createdAtColumnName,
				messagesTableName,
				chatIDColumnName,
				chatsTableName,
				idColumnName,
				chatsTableName,
				updatedAtColumnName,
				desc,
			),
		).
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

	var chats []domains.Chat

	for rows.Next() {
		chat := domains.Chat{}
		columns := pg.GetEntityColumns(&chat) // Only pointer to use rows.Scan() successfully
		columns = columns[:len(columns)-2]    // Not to paste Members, Messages fields to Scan function.

		err = rows.Scan(columns...)
		if err != nil {
			return nil, err
		}

		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}

func (repo *Repository) CreateChat(
	ctx context.Context,
	chat domains.Chat,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(chatsTableName).
		Columns(
			titleColumnName,
			descriptionColumnName,
			typeColumnName,
		).
		Values(
			chat.Title,
			chat.Description,
			chat.Type,
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
	if err != nil {
		return 0, err
	}

	var chatID uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&chatID); err != nil {
		return 0, err
	}

	builder := sq.Insert(chatsMembersTableName).
		Columns(chatIDColumnName, userIDColumnName)

	for _, member := range chat.Members {
		builder = builder.Values(
			chatID,
			member.ID,
		)
	}

	stmt, params, err = builder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return 0, err
	}

	if _, err = repo.tx.ExecContext(ctx, stmt, params...); err != nil {
		return 0, err
	}

	return chatID, nil
}

func (repo *Repository) GetChatByID(
	ctx context.Context,
	id uint64,
) (*domains.Chat, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(chatsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	chat := &domains.Chat{}

	columns := pg.GetEntityColumns(chat) // Only pointer to use rows.Scan() successfully

	columns = columns[:len(columns)-3] // Not to paste Members, Messages and IsRead fields to Scan function.

	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return chat, nil
}

func (repo *Repository) PrivateChatExists(
	ctx context.Context,
	members []domains.User,
) (bool, error) {
	builder := sq.
		Select(selectExists).
		From(chatsTableName).
		Where(
			sq.Eq{
				fmt.Sprintf("%s.%s", chatsTableName, typeColumnName): domains.ChatTypePrivate,
			},
		)

	for i, member := range members {
		tableAlias := fmt.Sprintf("%s%d", chatsMembersTableName, i)
		subquery := sq.
			Select(selectExists).
			From(fmt.Sprintf("%s %s", chatsMembersTableName, tableAlias)).
			Where(
				sq.And{
					sq.Expr(
						fmt.Sprintf(
							"%s.%s = %s.%s",
							tableAlias,
							chatIDColumnName,
							chatsTableName,
							idColumnName,
						),
					),
					sq.Eq{
						fmt.Sprintf("%s.%s", tableAlias, userIDColumnName): member.ID,
					},
				},
			)

		builder = builder.
			Where(
				sq.Expr(existsSubquery, subquery),
			)
	}

	stmt, params, err := builder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return false, err
	}

	var exists bool
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // Запись не найдена
		}

		return false, err
	}

	return exists, nil
}
