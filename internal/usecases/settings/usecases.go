package settings

import (
	"context"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	settingsService interfaces.SettingsService
}

func New(settingsService interfaces.SettingsService) *UseCases {
	return &UseCases{settingsService: settingsService}
}

func (u *UseCases) GetSettingsByUserID(
	ctx context.Context,
	userID uint64,
) (*domains.Settings, error) {
	return u.settingsService.GetSettingsByUserID(ctx, userID)
}

func (u *UseCases) UpdateSettings(
	ctx context.Context,
	settings domains.Settings,
) (*domains.Settings, error) {
	return u.settingsService.UpdateSettings(ctx, settings)
}
