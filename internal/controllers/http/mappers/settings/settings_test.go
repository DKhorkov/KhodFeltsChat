package settings_test

import (
	"testing"
	"time"

	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/settings"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
)

func TestMapSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		settings      domains.Settings
		expectedTheme int
	}{
		{
			name: "light theme",
			settings: domains.Settings{
				ID:        1,
				UserID:    10,
				Theme:     domains.ThemeLight,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectedTheme: 0,
		},
		{
			name: "dark theme",
			settings: domains.Settings{
				ID:        2,
				UserID:    20,
				Theme:     domains.ThemeDark,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			expectedTheme: 1,
		},
		{
			name:          "zero value settings",
			settings:      domains.Settings{},
			expectedTheme: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := mappers.MapSettings(tt.settings)

			assert.Equal(t, tt.expectedTheme, result.Theme)
		})
	}
}
