package settings

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	pg "github.com/DKhorkov/libs/db/postgresql"
)

type Service struct {
	uow                       interfaces.UnitOfWork
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository
}

func New(
	uow interfaces.UnitOfWork,
	newSettingsRepositoryFunc func(tx pg.Transaction) interfaces.SettingsRepository,
) *Service {
	return &Service{
		uow:                       uow,
		newSettingsRepositoryFunc: newSettingsRepositoryFunc,
	}
}

func (s *Service) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	var (
		settings *domains.Settings
		err      error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			settingsRepository := s.newSettingsRepositoryFunc(tx)
			if settings, err = settingsRepository.GetSettingsByUserID(ctx, userID); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", customerrors.ErrSettingsNotFound, err)
	}

	return settings, nil
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	settingsData domains.Settings,
) (*domains.Settings, error) {
	var (
		settings *domains.Settings
		err      error
	)

	err = s.uow.Do(
		ctx,
		func(ctx context.Context, tx pg.Transaction) error {
			settingsRepository := s.newSettingsRepositoryFunc(tx)

			if err = settingsRepository.UpdateSettings(ctx, settingsData); err != nil {
				return err
			}

			if settings, err = settingsRepository.GetSettingsByUserID(
				ctx,
				settingsData.UserID,
			); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return settings, nil
}
