package settings_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/settings"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_GetSettingsByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockSettingsRepository func(*mockrepositories.MockSettingsRepository)
	}

	type args struct {
		ctx    context.Context
		userID uint64
	}

	now := time.Now()

	expectedSettings := &domains.Settings{
		ID:        1,
		UserID:    10,
		Theme:     domains.ThemeLight,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Settings
		wantErr bool
		err     error
	}{
		{
			name: "successfully get settings",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(10)).
						Return(expectedSettings, nil)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
			},
			want:    expectedSettings,
			wantErr: false,
		},
		{
			name: "settings not found",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(999)).
						Return(nil, sql.ErrNoRows)
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrSettingsNotFound,
		},
		{
			name: "database error",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(10)).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 10,
			},
			want:    nil,
			wantErr: true,
			err:     customerrors.ErrSettingsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockSettingsRepo := mockrepositories.NewMockSettingsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockSettingsRepository != nil {
				tt.fields.mockSettingsRepository(mockSettingsRepo)
			}

			newSettingsRepoFunc := func(_ pg.Transaction) interfaces.SettingsRepository {
				return mockSettingsRepo
			}

			s := service.New(mockUOW, newSettingsRepoFunc)

			// Act
			got, err := s.GetSettingsByUserID(tt.args.ctx, tt.args.userID)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_UpdateSettings(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockUOW                func(*mockuow.MockUnitOfWork)
		mockSettingsRepository func(*mockrepositories.MockSettingsRepository)
	}

	type args struct {
		ctx          context.Context
		settingsData domains.Settings
	}

	now := time.Now()

	updatedSettings := &domains.Settings{
		ID:        1,
		UserID:    10,
		Theme:     domains.ThemeDark,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domains.Settings
		wantErr bool
		err     error
	}{
		{
			name: "successfully update settings",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						UpdateSettings(gomock.Any(), domains.Settings{UserID: 10, Theme: domains.ThemeDark}).
						Return(nil)

					sr.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(10)).
						Return(updatedSettings, nil)
				},
			},
			args: args{
				ctx:          context.Background(),
				settingsData: domains.Settings{UserID: 10, Theme: domains.ThemeDark},
			},
			want:    updatedSettings,
			wantErr: false,
		},
		{
			name: "error updating settings",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						UpdateSettings(gomock.Any(), gomock.Any()).
						Return(errors.New("database error"))
				},
			},
			args: args{
				ctx:          context.Background(),
				settingsData: domains.Settings{UserID: 10, Theme: domains.ThemeDark},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
		{
			name: "error getting settings after update",
			fields: fields{
				mockUOW: func(uow *mockuow.MockUnitOfWork) {
					uow.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
							tx := &struct{ pg.Transaction }{}

							return fn(ctx, tx)
						})
				},
				mockSettingsRepository: func(sr *mockrepositories.MockSettingsRepository) {
					sr.EXPECT().
						UpdateSettings(gomock.Any(), gomock.Any()).
						Return(nil)

					sr.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(10)).
						Return(nil, errors.New("not found"))
				},
			},
			args: args{
				ctx:          context.Background(),
				settingsData: domains.Settings{UserID: 10, Theme: domains.ThemeDark},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockUOW := mockuow.NewMockUnitOfWork(ctrl)
			mockSettingsRepo := mockrepositories.NewMockSettingsRepository(ctrl)

			if tt.fields.mockUOW != nil {
				tt.fields.mockUOW(mockUOW)
			}

			if tt.fields.mockSettingsRepository != nil {
				tt.fields.mockSettingsRepository(mockSettingsRepo)
			}

			newSettingsRepoFunc := func(_ pg.Transaction) interfaces.SettingsRepository {
				return mockSettingsRepo
			}

			s := service.New(mockUOW, newSettingsRepoFunc)

			// Act
			got, err := s.UpdateSettings(tt.args.ctx, tt.args.settingsData)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)

				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
