package web_push_subscriptions

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                                   pg.UnitOfWork
	newWebPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.WebPushSubscriptionsRepository
}

func New(
	uow pg.UnitOfWork,
	newWebPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.WebPushSubscriptionsRepository,
) *Service {
	return &Service{
		uow:                                   uow,
		newWebPushSubscriptionsRepositoryFunc: newWebPushSubscriptionsRepositoryFunc,
	}
}

func (s *Service) CreateWebPushSubscription(
	ctx context.Context,
	subscription domains.WebPushSubscription,
) (*domains.WebPushSubscription, error) {
	var (
		result *domains.WebPushSubscription
		err    error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newWebPushSubscriptionsRepositoryFunc(tx)

			id, createErr := repo.CreateWebPushSubscription(ctx, subscription)
			if createErr != nil {
				return createErr
			}

			subscriptions, getErr := repo.GetWebPushSubscriptionsByUserID(ctx, subscription.UserID)
			if getErr != nil {
				return getErr
			}

			for _, sub := range subscriptions {
				if sub.ID == id {
					result = &sub

					return nil
				}
			}

			return fmt.Errorf("%w: id=%d", customerrors.ErrWebPushSubscriptionNotFound, id)
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) GetWebPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.WebPushSubscription, error) {
	var (
		subscriptions []domains.WebPushSubscription
		err           error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newWebPushSubscriptionsRepositoryFunc(tx)
			if subscriptions, err = repo.GetWebPushSubscriptionsByUserID(ctx, userID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *Service) DeleteWebPushSubscription(
	ctx context.Context,
	id uint64,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newWebPushSubscriptionsRepositoryFunc(tx)

			return repo.DeleteWebPushSubscription(ctx, id)
		},
	)
}
