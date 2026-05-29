package users

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                    interfaces.UnitOfWork
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository
}

func New(
	uow interfaces.UnitOfWork,
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository,
) *Service {
	return &Service{
		uow:                    uow,
		newUsersRepositoryFunc: newUsersRepositoryFunc,
	}
}

func (s *Service) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) ([]domains.User, error) {
	var (
		users []domains.User
		err   error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if users, err = usersRepository.GetUsers(ctx, filters, pagination); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *Service) GetUserByID(ctx context.Context, id uint64) (*domains.User, error) {
	var (
		user *domains.User
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if user, err = usersRepository.GetUserByID(ctx, id); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrUserNotFound, err)
	}

	return user, nil
}

func (s *Service) GetUserByEmail(
	ctx context.Context,
	email string,
) (*domains.User, error) {
	var (
		user *domains.User
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if user, err = usersRepository.GetUserByEmail(ctx, email); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrUserNotFound, err)
	}

	return user, nil
}

func (s *Service) GetUserByUsername(
	ctx context.Context,
	username string,
) (*domains.User, error) {
	var (
		user *domains.User
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if user, err = usersRepository.GetUserByUsername(ctx, username); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrUserNotFound, err)
	}

	return user, nil
}

func (s *Service) UpdateUser(
	ctx context.Context,
	user domains.User,
) (*domains.User, error) {
	var (
		updatedUser *domains.User
		err         error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)

			err = usersRepository.UpdateUser(ctx, user)
			if err != nil {
				return err
			}

			if updatedUser, err = usersRepository.GetUserByID(ctx, user.ID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}
