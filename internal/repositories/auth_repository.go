package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/DKhorkov/libs/db/postgresql"

	sq "github.com/Masterminds/squirrel"

	"github.com/DKhorkov/kfc/internal/domains"
)

const (
	refreshTokensTableName      = "refresh_tokens"
	refreshTokenValueColumnName = "value"
	refreshTokenTTLColumnName   = "ttl"
	returningIDSuffix           = "RETURNING id"
	userIDColumnName            = "user_id"
	emailConfirmedColumnName    = "email_confirmed"
	passwordColumnName          = "password"
)

type AuthRepository struct {
	tx *sql.Tx
}

func NewAuthRepository(
	tx *sql.Tx,
) *AuthRepository {
	return &AuthRepository{
		tx: tx,
	}
}

func (repo *AuthRepository) RegisterUser(
	ctx context.Context,
	userData domains.RegisterDTO,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(usersTableName).
		Columns(
			usernameColumnName,
			emailColumnName,
			passwordColumnName,
		).
		Values(
			userData.Username,
			userData.Email,
			userData.Password,
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
	if err != nil {
		return 0, err
	}

	var userID uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&userID); err != nil {
		return 0, err
	}

	return userID, nil
}

func (repo *AuthRepository) CreateRefreshToken(
	ctx context.Context,
	userID uint64,
	refreshToken string,
	ttl time.Duration,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(refreshTokensTableName).
		Columns(
			userIDColumnName,
			refreshTokenValueColumnName,
			refreshTokenTTLColumnName,
		).
		Values(
			userID,
			refreshToken,
			time.Now().UTC().Add(ttl),
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
	if err != nil {
		return 0, err
	}

	var refreshTokenID uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&refreshTokenID); err != nil {
		return 0, err
	}

	return refreshTokenID, nil
}

func (repo *AuthRepository) GetRefreshTokenByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.RefreshToken, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(refreshTokensTableName).
		Where(sq.Eq{userIDColumnName: userID}).
		Where(
			sq.Expr(
				refreshTokenTTLColumnName + " > CURRENT_TIMESTAMP",
			),
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	refreshToken := &domains.RefreshToken{}

	columns := postgresql.GetEntityColumns(refreshToken)
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (repo *AuthRepository) ExpireRefreshToken(ctx context.Context, refreshToken string) error {
	stmt, params, err := sq.
		Update(refreshTokensTableName).
		Where(sq.Eq{refreshTokenValueColumnName: refreshToken}).
		Set(
			refreshTokenTTLColumnName,
			time.Now().UTC().Add(time.Hour*time.Duration(-24)),
		).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
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

func (repo *AuthRepository) VerifyEmail(ctx context.Context, userID uint64) error {
	stmt, params, err := sq.
		Update(usersTableName).
		Where(sq.Eq{idColumnName: userID}).
		Set(emailConfirmedColumnName, true).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
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

func (repo *AuthRepository) ChangePassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	stmt, params, err := sq.
		Update(usersTableName).
		Where(sq.Eq{idColumnName: userID}).
		Set(passwordColumnName, newPassword).
		PlaceholderFormat(sq.Dollar). // pq postgres driver works only with $ placeholders
		ToSql()
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
