package push_subscriptions

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                                interfaces.UnitOfWork
	newPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.PushSubscriptionsRepository
}

func New(
	uow interfaces.UnitOfWork,
	newPushSubscriptionsRepositoryFunc func(tx pg.Transaction) interfaces.PushSubscriptionsRepository,
) *Service {
	return &Service{
		uow:                                uow,
		newPushSubscriptionsRepositoryFunc: newPushSubscriptionsRepositoryFunc,
	}
}

func (s *Service) CreatePushSubscription(
	ctx context.Context,
	subscription domains.PushSubscription,
) (*domains.PushSubscription, error) {
	var (
		result *domains.PushSubscription
		err    error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)

			id, createErr := repo.CreatePushSubscription(ctx, subscription)
			if createErr != nil {
				return createErr
			}

			subscriptions, getErr := repo.GetPushSubscriptionsByUserID(ctx, subscription.UserID)
			if getErr != nil {
				return getErr
			}

			for _, sub := range subscriptions {
				if sub.ID == id {
					result = &sub

					return nil
				}
			}

			return fmt.Errorf("%w: id=%d", customerrors.ErrPushSubscriptionNotFound, id)
		},
	)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) GetPushSubscriptionsByUserID(
	ctx context.Context,
	userID uint64,
) ([]domains.PushSubscription, error) {
	var (
		subscriptions []domains.PushSubscription
		err           error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)
			if subscriptions, err = repo.GetPushSubscriptionsByUserID(ctx, userID); err != nil {
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

func (s *Service) DeletePushSubscription(
	ctx context.Context,
	id uint64,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			repo := s.newPushSubscriptionsRepositoryFunc(tx)

			return repo.DeletePushSubscription(ctx, id)
		},
	)
}
