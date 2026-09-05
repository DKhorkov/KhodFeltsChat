package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
	customnats "github.com/DKhorkov/libs/nats"
)

type Service struct {
	uow                       pg.UnitOfWork
	newAuthRepositoryFunc     func(tx pg.Transaction) interfaces.AuthRepository
	newUsersRepositoryFunc    func(tx pg.Transaction) interfaces.UsersRepository
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository
	natsPublisher             customnats.Publisher
	natsConfig                config.NATSConfig
}

func New(
	uow pg.UnitOfWork,
	newAuthRepositoryFunc func(tx pg.Transaction) interfaces.AuthRepository,
	newUsersRepositoryFunc func(tx pg.Transaction) interfaces.UsersRepository,
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository,
	natsPublisher customnats.Publisher,
	natsConfig config.NATSConfig,
) *Service {
	return &Service{
		uow:                       uow,
		newAuthRepositoryFunc:     newAuthRepositoryFunc,
		newUsersRepositoryFunc:    newUsersRepositoryFunc,
		newSettingsRepositoryFunc: newSettingsRepositoryFunc,
		natsPublisher:             natsPublisher,
		natsConfig:                natsConfig,
	}
}

func (s *Service) RegisterUser(
	ctx context.Context,
	userData domains.RegisterDTO,
) (*domains.User, error) {
	var (
		user *domains.User
		err  error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)
			if user, _ = usersRepository.GetUserByEmail(ctx, userData.Email); user != nil {
				return fmt.Errorf(
					"%w: user with provided email already exists",
					customerrors.ErrUserAlreadyExists,
				)
			}

			if user, _ = usersRepository.GetUserByUsername(ctx, userData.Username); user != nil {
				return fmt.Errorf(
					"%w: user with provided username already exists",
					customerrors.ErrUserAlreadyExists,
				)
			}

			authRepository := s.newAuthRepositoryFunc(tx)
			if _, err = authRepository.RegisterUser(ctx, userData); err != nil {
				return err
			}

			user, err = usersRepository.GetUserByEmail(ctx, userData.Email)
			if err != nil {
				return err
			}

			settingsRepository := s.newSettingsRepositoryFunc(tx)
			if err = settingsRepository.CreateSettings(ctx, domains.Settings{
				UserID:          user.ID,
				Theme:           domains.ThemeLight,
				EmailConsents:   domains.ConsentNewMessage,
				WebPushConsents: domains.ConsentNewMessage,
			}); err != nil {
				return err
			}

			emailDTO := &domains.EmailNotificationDTO{
				Type:    domains.EmailTypeVerifyEmail,
				UserID:  user.ID,
				Payload: nil,
			}

			content, err := json.Marshal(emailDTO) //nolint:govet // неважное затенение
			if err != nil {
				return err
			}

			if err = s.natsPublisher.Publish(
				s.natsConfig.Subjects.EmailNotification,
				content,
			); err != nil {
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

func (s *Service) CreateRefreshToken(
	ctx context.Context,
	userID uint64,
	value string,
	ttl time.Duration,
) (*domains.RefreshToken, error) {
	var (
		refreshToken *domains.RefreshToken
		err          error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
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

			if refreshToken, err = authRepository.GetRefreshTokenByValue(ctx, value); err != nil {
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

func (s *Service) GetRefreshTokenByValue(
	ctx context.Context,
	value string,
) (*domains.RefreshToken, error) {
	var (
		refreshToken *domains.RefreshToken
		err          error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if refreshToken, err = authRepository.GetRefreshTokenByValue(ctx, value); err != nil {
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

func (s *Service) ExpireRefreshToken(ctx context.Context, refreshTokenID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			return authRepository.ExpireRefreshToken(ctx, refreshTokenID)
		},
	)
}

func (s *Service) ExpireAllUserRefreshTokens(ctx context.Context, userID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			return authRepository.ExpireAllUserRefreshTokens(ctx, userID)
		},
	)
}

func (s *Service) VerifyEmail(ctx context.Context, userID uint64) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			return authRepository.VerifyEmail(ctx, userID)
		},
	)
}

func (s *Service) ForgetPassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)
			if err := authRepository.ChangePassword(ctx, userID, newPassword); err != nil {
				return err
			}

			return authRepository.ExpireAllUserRefreshTokens(ctx, userID)
		},
	)
}

func (s *Service) ChangePassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			authRepository := s.newAuthRepositoryFunc(tx)

			return authRepository.ChangePassword(ctx, userID, newPassword)
		},
	)
}

func (s *Service) SendForgetPasswordMessage(ctx context.Context, email string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)

			user, err := usersRepository.GetUserByEmail(ctx, email)
			if err != nil {
				return fmt.Errorf("%w: %w", customerrors.ErrUserNotFound, err)
			}

			emailDTO := &domains.EmailNotificationDTO{
				Type:    domains.EmailTypeForgetPassword,
				UserID:  user.ID,
				Payload: nil,
			}

			content, err := json.Marshal(emailDTO)
			if err != nil {
				return err
			}

			if err = s.natsPublisher.Publish(
				s.natsConfig.Subjects.EmailNotification,
				content,
			); err != nil {
				return err
			}

			return nil
		},
	)
}

func (s *Service) SendVerifyEmailMessage(ctx context.Context, email string) error {
	return s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			usersRepository := s.newUsersRepositoryFunc(tx)

			user, err := usersRepository.GetUserByEmail(ctx, email)
			if err != nil {
				return fmt.Errorf("%w: %w", customerrors.ErrUserNotFound, err)
			}

			emailDTO := &domains.EmailNotificationDTO{
				Type:    domains.EmailTypeVerifyEmail,
				UserID:  user.ID,
				Payload: nil,
			}

			content, err := json.Marshal(emailDTO)
			if err != nil {
				return err
			}

			if err = s.natsPublisher.Publish(
				s.natsConfig.Subjects.EmailNotification,
				content,
			); err != nil {
				return err
			}

			return nil
		},
	)
}
