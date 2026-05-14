package settings_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/usecases/settings"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_GetSettingsByUserID(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockSettingsService func(*mockservices.MockSettingsService)
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
				mockSettingsService: func(ss *mockservices.MockSettingsService) {
					ss.EXPECT().
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
			name: "service returns error",
			fields: fields{
				mockSettingsService: func(ss *mockservices.MockSettingsService) {
					ss.EXPECT().
						GetSettingsByUserID(gomock.Any(), uint64(999)).
						Return(nil, errors.New("settings not found"))
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: 999,
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("settings not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockSettingsService := mockservices.NewMockSettingsService(ctrl)

			if tt.fields.mockSettingsService != nil {
				tt.fields.mockSettingsService(mockSettingsService)
			}

			u := settings.New(mockSettingsService)

			// Act
			got, err := u.GetSettingsByUserID(tt.args.ctx, tt.args.userID)

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

func TestUseCases_UpdateSettings(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockSettingsService func(*mockservices.MockSettingsService)
	}

	type args struct {
		ctx      context.Context
		settings domains.Settings
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
				mockSettingsService: func(ss *mockservices.MockSettingsService) {
					ss.EXPECT().
						UpdateSettings(gomock.Any(), domains.Settings{UserID: 10, Theme: domains.ThemeDark}).
						Return(updatedSettings, nil)
				},
			},
			args: args{
				ctx:      context.Background(),
				settings: domains.Settings{UserID: 10, Theme: domains.ThemeDark},
			},
			want:    updatedSettings,
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockSettingsService: func(ss *mockservices.MockSettingsService) {
					ss.EXPECT().
						UpdateSettings(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("database error"))
				},
			},
			args: args{
				ctx:      context.Background(),
				settings: domains.Settings{UserID: 10, Theme: domains.ThemeDark},
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			ctrl := gomock.NewController(t)

			mockSettingsService := mockservices.NewMockSettingsService(ctrl)

			if tt.fields.mockSettingsService != nil {
				tt.fields.mockSettingsService(mockSettingsService)
			}

			u := settings.New(mockSettingsService)

			// Act
			got, err := u.UpdateSettings(tt.args.ctx, tt.args.settings)

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
