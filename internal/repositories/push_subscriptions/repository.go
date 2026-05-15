package push_subscriptions

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	pg "github.com/DKhorkov/libs/db/postgresql"
	sq "github.com/Masterminds/squirrel"
)

const (
	pushSubscriptionsTableName = "push_subscriptions"

	idColumnName        = "id"
	userIDColumnName    = "user_id"
	endpointColumnName  = "endpoint"
	p256dhColumnName    = "p256dh"
	authColumnName      = "auth"
	createdAtColumnName = "created_at"

	selectAllColumns = "*"
)

type Repository struct {
	tx pg.Transaction
}

func New(tx pg.Transaction) *Repository {
	return &Repository{tx: tx}
}

func (repo *Repository) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (uint64, error) {
	stmt, params, err := sq.
		Insert(pushSubscriptionsTableName).
		Columns(
			userIDColumnName,
			endpointColumnName,
			p256dhColumnName,
			authColumnName,
		).
		Values(
			subscription.UserID,
			subscription.Endpoint,
			subscription.P256dh,
			subscription.Auth,
		).
		Suffix("RETURNING " + idColumnName).
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

func (repo *Repository) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	stmt, params, err := sq.
		Select(selectAllColumns).
		From(pushSubscriptionsTableName).
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

	var subscriptions []domains.PushSubscription

	for rows.Next() {
		subscription := domains.PushSubscription{}
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

func (repo *Repository) DeletePushSubscription(ctx context.Context, id uint64) error {
	stmt, params, err := sq.
		Delete(pushSubscriptionsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}

func (repo *Repository) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	stmt, params, err := sq.
		Delete(pushSubscriptionsTableName).
		Where(sq.Eq{endpointColumnName: endpoint}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = repo.tx.ExecContext(ctx, stmt, params...)

	return err
}
