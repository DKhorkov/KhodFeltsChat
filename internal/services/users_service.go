package services

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type UsersService struct {
	uow                    interfaces.UnitOfWork
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository
}

func NewUsersService(
	uow interfaces.UnitOfWork,
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository,
) *UsersService {
	return &UsersService{
		uow:                    uow,
		newUsersRepositoryFunc: newUsersRepositoryFunc,
	}
}

func (s *UsersService) GetUsers(
	ctx context.Context,
	filters *domains.UsersFilters,
	pagination *domains.Pagination,
) (users []domains.User, err error) {
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

func (s *UsersService) GetUserByID(ctx context.Context, id uint64) (user *domains.User, err error) {
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

func (s *UsersService) GetUserByEmail(
	ctx context.Context,
	email string,
) (user *domains.User, err error) {
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

func (s *UsersService) GetUserByUsername(
	ctx context.Context,
	username string,
) (user *domains.User, err error) {
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

func (s *UsersService) UpdateUser(
	ctx context.Context,
	userData domains.UpdateUserDTO,
) (user *domains.User, err error) {
	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)

			err = usersRepository.UpdateUser(ctx, userData)
			if err != nil {
				return err
			}

			if user, err = usersRepository.GetUserByID(ctx, userData.ID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}
