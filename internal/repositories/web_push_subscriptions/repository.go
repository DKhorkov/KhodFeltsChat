package web_push_subscriptions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	webPushSubscriptionsTableName = "web_push_subscriptions"

	idColumnName            = "id"
	userIDColumnName        = "user_id"
	endpointColumnName      = "endpoint"
	encryptionKeyColumnName = "encryption_key"
	authColumnName          = "auth"
	createdAtColumnName     = "created_at"

	selectAllColumns = "*"

	returningIDSuffix = "RETURNING id"
)

type Repository struct {
	tx pg.Transaction
}

func New(tx pg.Transaction) *Repository {
	return &Repository{tx: tx}
}

func (repo *Repository) CreateWebPushSubscription(
	ctx context.Context,
	subscription domains.WebPushSubscription,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(webPushSubscriptionsTableName).
		Columns(
			userIDColumnName,
			endpointColumnName,
			encryptionKeyColumnName,
			authColumnName,
		).
		Values(
			subscription.UserID,
			subscription.Endpoint,
			subscription.EncryptionKey,
			subscription.Auth,
		).
		Suffix(returningIDSuffix).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return 0, err
	}

	var id uint64
	if err = repo.tx.QueryRowContext(ctx, stmt, params...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (repo *Repository) GetWebPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.WebPushSubscription, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(webPushSubscriptionsTableName).
		Where(sq.Eq{userIDColumnName: userID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := repo.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subscriptions []domains.WebPushSubscription

	for rows.Next() {
		subscription := domains.WebPushSubscription{}
		columns := pg.GetEntityColumns(&subscription)

		if err = rows.Scan(columns...); err != nil {
			return nil, err
		}

		subscriptions = append(subscriptions, subscription)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (repo *Repository) DeleteWebPushSubscription(ctx context.Context, id uint64) error {
	stmt, params, err := sq.
		Delete(webPushSubscriptionsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
