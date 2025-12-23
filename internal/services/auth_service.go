package services

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"time"
)

type AuthService struct {
	uow                     interfaces.UnitOfWork
	newAuthRepositoryFunc   func(tx *sql.Tx) interfaces.AuthRepository
	newUsersRepositoryFunc  func(tx *sql.Tx) interfaces.UsersRepository
	newEmailsRepositoryFunc func() interfaces.EmailsRepository
}

func NewAuthService(
	uow interfaces.UnitOfWork,
	newAuthRepositoryFunc func(tx *sql.Tx) interfaces.AuthRepository,
	newUsersRepositoryFunc func(tx *sql.Tx) interfaces.UsersRepository,
	newEmailsRepositoryFunc func() interfaces.EmailsRepository,
) *AuthService {
	return &AuthService{
		uow:                     uow,
		newAuthRepositoryFunc:   newAuthRepositoryFunc,
		newUsersRepositoryFunc:  newUsersRepositoryFunc,
		newEmailsRepositoryFunc: newEmailsRepositoryFunc,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, userData domains.RegisterDTO) (user *domains.User, err error) {
	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if user, _ = usersRepository.GetUserByEmail(ctx, userData.Email); user != nil {
				return fmt.Errorf("%w: user with provided email already exists", customerrors.ErrUserAlreadyExists)
			}

			if user, _ = usersRepository.GetUserByUsername(ctx, userData.Email); user != nil {
				return fmt.Errorf("%w: user with provided username already exists", customerrors.ErrUserAlreadyExists)
			}

			authRepository := s.newAuthRepositoryFunc(tx)
			if _, err = authRepository.RegisterUser(ctx, userData); err != nil {
				return err
			}

			user, err = usersRepository.GetUserByEmail(ctx, userData.Email)
			if err != nil {
				return err
			}

			emailsRepository := s.newEmailsRepositoryFunc()
			if err = emailsRepository.SendVerifyEmailMessage(ctx, *user); err != nil {
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

func (s *AuthService) CreateRefreshToken(
	ctx context.Context,
	userID uint64,
	value string,
	ttl time.Duration,
) (refreshToken *domains.RefreshToken, err error) {
	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			_, err = authRepository.CreateRefreshToken(
				ctx,
				userID,
				value,
				ttl,
			)

			if err != nil {
				return err
			}

			if refreshToken, err = authRepository.GetRefreshTokenByUserID(ctx, userID); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (s *AuthService) GetRefreshTokenByUserID(
	ctx context.Context,
	userID uint64,
) (refreshToken *domains.RefreshToken, err error) {
	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if refreshToken, err = authRepository.GetRefreshTokenByUserID(ctx, userID); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return refreshToken, nil
}

func (s *AuthService) ExpireRefreshToken(ctx context.Context, refreshToken string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			return authRepository.ExpireRefreshToken(ctx, refreshToken)
		},
	)
}

func (s *AuthService) VerifyEmail(ctx context.Context, userID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			return authRepository.VerifyEmail(ctx, userID)
		},
	)
}

func (s *AuthService) ForgetPassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if err := authRepository.ChangePassword(ctx, userID, newPassword); err != nil {
				return err
			}

			refreshToken, err := authRepository.GetRefreshTokenByUserID(ctx, userID)
			if err != nil {
				return err
			}

			return authRepository.ExpireRefreshToken(ctx, refreshToken.Value)
		},
	)
}

func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			return authRepository.ChangePassword(ctx, userID, newPassword)
		},
	)
}

func (s *AuthService) SendForgetPasswordMessage(ctx context.Context, email string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			emailsRepository := s.newEmailsRepositoryFunc()

			user, err := usersRepository.GetUserByEmail(ctx, email)
			if err != nil {
				return err
			}

			return emailsRepository.SendForgetPasswordMessage(ctx, *user)
		},
	)
}

func (s *AuthService) SendVerifyEmailMessage(ctx context.Context, email string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			emailsRepository := s.newEmailsRepositoryFunc()

			user, err := usersRepository.GetUserByEmail(ctx, email)
			if err != nil {
				return err
			}

			return emailsRepository.SendVerifyEmailMessage(ctx, *user)
		},
	)
}
