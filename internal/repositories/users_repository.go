package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/DKhorkov/libs/db/postgresql"

	sq "github.com/Masterminds/squirrel"

	"github.com/DKhorkov/kfc/internal/domains"
)

const (
	selectAllColumns    = "*"
	usersTableName      = "users"
	idColumnName        = "id"
	usernameColumnName  = "username"
	emailColumnName     = "email"
	createdAtColumnName = "created_at"
	updatedAtColumnName = "updated_at"
	desc                = "DESC"
	asc                 = "ASC"
)

type UsersRepository struct {
	tx *sql.Tx
}

func NewUsersRepository(
	tx *sql.Tx,
) *UsersRepository {
	return &UsersRepository{
		tx: tx,
	}
}

func (repo *UsersRepository) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
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

	columns := postgresql.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *UsersRepository) GetUserByUsername(
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

	columns := postgresql.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *UsersRepository) GetUserByEmail(
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

	columns := postgresql.GetEntityColumns(user)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return user, nil
}

func (repo *UsersRepository) GetUsers(
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
		rowsErr := rows.Close()
		if rowsErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; %w", err, rowsErr)

				return
			}

			err = rowsErr
		}
	}()

	var users []domains.User

	for rows.Next() {
		user := domains.User{}
		columns := postgresql.GetEntityColumns(&user) // Only pointer to use rows.Scan() successfully

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

func (repo *UsersRepository) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) error {
	builder := sq.
		Update(usersTableName).
		Where(sq.Eq{idColumnName: userData.ID}).
		Set(usernameColumnName, userData.Username).
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
