package reactions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                        pg.UnitOfWork
	newReactionsRepositoryFunc func(tx pg.Transaction) interfaces.ReactionsRepository
}

func New(
	uow pg.UnitOfWork,
	newReactionsRepositoryFunc func(tx pg.Transaction) interfaces.ReactionsRepository,
) *Service {
	return &Service{
		uow:                        uow,
		newReactionsRepositoryFunc: newReactionsRepositoryFunc,
	}
}

func (s *Service) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	var (
		reactions []domains.Reaction
		err       error
	)

	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		reactions, err = repo.ListReactions(ctx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return reactions, nil
}

func (s *Service) GetReactionByID(
	ctx context.Context,
	id uint64,
) (*domains.Reaction, error) {
	var (
		reaction *domains.Reaction
		err      error
	)

	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		reaction, err = repo.GetReactionByID(ctx, id)

		return err
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, customerrors.ErrReactionNotFound
		}

		return nil, err
	}

	return reaction, nil
}

func (s *Service) AddMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	return s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)

		return repo.AddMessageReaction(ctx, dto)
	})
}

func (s *Service) RemoveMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	return s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)

		return repo.RemoveMessageReaction(ctx, dto)
	})
}

func (s *Service) ListReactionsForMessages(
	ctx context.Context,
	messageIDs []uint64,
) (map[uint64][]domains.MessageReactionSummary, error) {
	var (
		result map[uint64][]domains.MessageReactionSummary
		err    error
	)

	err = s.uow.Do(ctx, func(ctx context.Context, tx pg.Transaction) error {
		repo := s.newReactionsRepositoryFunc(tx)
		result, err = repo.ListReactionsForMessages(ctx, messageIDs)

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
