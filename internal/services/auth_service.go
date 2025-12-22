package services

import (
	"context"
	"database/sql"
	"github.com/DKhorkov/khodfeltschat/internal/domains"
	customerrors "github.com/DKhorkov/khodfeltschat/internal/errors"
	"github.com/DKhorkov/khodfeltschat/internal/interfaces"
	"time"
)

type AuthService struct {
	uow                     interfaces.UnitOfWork
	newUsersRepositoryFunc  func() interfaces.UsersRepository
	newAuthRepositoryFunc   func() interfaces.AuthRepository
	newEmailsRepositoryFunc func() interfaces.EmailsRepository
}

func NewAuthService(
	uow interfaces.UnitOfWork,
	newUsersRepositoryFunc func() interfaces.UsersRepository,
	newAuthRepositoryFunc func() interfaces.AuthRepository,
	newEmailsRepositoryFunc func() interfaces.EmailsRepository,
) *AuthService {
	return &AuthService{
		uow:                     uow,
		newUsersRepositoryFunc:  newUsersRepositoryFunc,
		newAuthRepositoryFunc:   newAuthRepositoryFunc,
		newEmailsRepositoryFunc: newEmailsRepositoryFunc,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, userData domains.RegisterDTO) (user *domains.User, err error) {
	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			usersRepository := s.newUsersRepositoryFunc()
			if user, _ = usersRepository.GetUserByEmail(ctx, userData.Email); user != nil {
				return customerrors.UserAlreadyExistsError{}
			}

			if user, _ = usersRepository.GetUserByUsername(ctx, userData.Email); user != nil {
				return customerrors.UserAlreadyExistsError{Message: "user with provided username already exists"}
			}

			authRepository := s.newAuthRepositoryFunc()
			if _, err = authRepository.RegisterUser(ctx, userData); err != nil {
				return err
			}

			user, err = usersRepository.GetUserByEmail(ctx, userData.Email)
			if err != nil {
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
			authRepository := s.newAuthRepositoryFunc()
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
			authRepository := s.newAuthRepositoryFunc()
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
			authRepository := s.newAuthRepositoryFunc()
			return authRepository.ExpireRefreshToken(ctx, refreshToken)
		},
	)
}

func (s *AuthService) VerifyEmail(ctx context.Context, userID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			authRepository := s.newAuthRepositoryFunc()
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
			authRepository := s.newAuthRepositoryFunc()
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
			authRepository := s.newAuthRepositoryFunc()
			return authRepository.ChangePassword(ctx, userID, newPassword)
		},
	)
}

func (s *AuthService) SendForgetPasswordMessage(ctx context.Context, email string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx *sql.Tx) error {
			usersRepository := s.newUsersRepositoryFunc()
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
			usersRepository := s.newUsersRepositoryFunc()
			emailsRepository := s.newEmailsRepositoryFunc()

			user, err := usersRepository.GetUserByEmail(ctx, email)
			if err != nil {
				return err
			}

			return emailsRepository.SendVerifyEmailMessage(ctx, *user)
		},
	)
}
