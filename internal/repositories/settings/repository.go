package settings

import (
	"context"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	tableName = "settings"

	idColumnName        = "id"
	userIDColumnName    = "user_id"
	themeColumnName     = "theme"
	createdAtColumnName = "created_at"
	updatedAtColumnName = "updated_at"

	selectAllColumns = "*"
)

type Repository struct {
	tx pg.Transaction
}

func New(tx pg.Transaction) *Repository {
	return &Repository{tx: tx}
}

func (repo *Repository) CreateSettings(ctx context.Context, settings domains.Settings) error {
	stmt, params, err := sq.
		Insert(tableName).
		Columns(userIDColumnName, themeColumnName).
		Values(settings.UserID, settings.Theme).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(tableName).
		Where(sq.Eq{userIDColumnName: userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	settings := &domains.Settings{}
	columns := pg.GetEntityColumns(settings)

	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return settings, nil
}

func (repo *Repository) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) error {
	builder := sq.
		Update(tableName).
		Where(sq.Eq{userIDColumnName: settings.UserID}).
		Set(themeColumnName, settings.Theme).
		Set(updatedAtColumnName, time.Now()).
		PlaceholderFormat(sq.Dollar)

	stmt, params, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
