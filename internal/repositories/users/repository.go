package users

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/logging"
	sq "github.com/Masterminds/squirrel"
)

const (
	usersTableName = "users"

	idColumnName         = "id"
	usernameColumnName   = "username"
	emailColumnName      = "email"
	createdAtColumnName  = "created_at"
	updatedAtColumnName  = "updated_at"
	avatarPathColumnName = "avatar_path"

	desc = "DESC"
	asc  = "ASC"

	selectAllColumns = "*"
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

func (repo *Repository) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(usersTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	user := &domains.User{}

	columns := pg.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *Repository) GetUserByUsername(
	ctx context.Context,
	username string,
) (*domains.User, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(usersTableName).
		Where(sq.Eq{usernameColumnName: username}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	user := &domains.User{}

	columns := pg.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*domains.User, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(usersTableName).
		Where(sq.Eq{emailColumnName: email}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	user := &domains.User{}

	columns := pg.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *Repository) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	builder := sq.
		Select(selectAllColumns).
		From(usersTableName).
		OrderBy(fmt.Sprintf("%s %s", idColumnName, desc)).
		PlaceholderFormat(sq.Dollar)

	if filters != nil && filters.Username != nil && *filters.Username != "" {
		searchTerm := "%" + strings.ToLower(*filters.Username) + "%"
		builder = builder.
			Where(
				sq.ILike{
					fmt.Sprintf(
						"%s.%s",
						usersTableName,
						usernameColumnName,
					): searchTerm,
				},
			)
	}

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

func (repo *Repository) UpdateUser(
	ctx context.Context,
	user domains.User,
) error {
	builder := sq.
		Update(usersTableName).
		Where(sq.Eq{idColumnName: user.ID}).
		Set(usernameColumnName, user.Username).
		Set(emailColumnName, user.Email).
		Set(avatarPathColumnName, user.AvatarPath).
		Set(updatedAtColumnName, time.Now()).
		PlaceholderFormat(sq.Dollar) // pq postgres driver works only with $ placeholders

	stmt, params, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(
		ctx,
		stmt,
		params...,
	)

	return err
}
