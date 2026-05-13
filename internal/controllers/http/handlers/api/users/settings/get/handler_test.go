package get_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/settings/get"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	createRequest := func(t *testing.T, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/api/users/me/settings", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	t.Run("successful get settings", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := get.Handler(mockUseCase)

		userID := uint64(123)
		now := time.Now()
		expectedSettings := &domains.Settings{
			ID:        1,
			UserID:    userID,
			Theme:     domains.ThemeLight,
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockUseCase.EXPECT().
			GetSettingsByUserID(gomock.Any(), userID).
			Return(expectedSettings, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(0), response["theme"])
	})

	t.Run("successful get dark theme settings", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := get.Handler(mockUseCase)

		userID := uint64(456)
		now := time.Now()
		expectedSettings := &domains.Settings{
			ID:        2,
			UserID:    userID,
			Theme:     domains.ThemeDark,
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockUseCase.EXPECT().
			GetSettingsByUserID(gomock.Any(), userID).
			Return(expectedSettings, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["theme"])
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := get.Handler(mockUseCase)

		req := httptest.NewRequest(http.MethodGet, "/api/users/me/settings", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("settings not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := get.Handler(mockUseCase)

		userID := uint64(999)

		mockUseCase.EXPECT().
			GetSettingsByUserID(gomock.Any(), userID).
			Return(nil, customerrors.ErrSettingsNotFound)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("internal server error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := get.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			GetSettingsByUserID(gomock.Any(), userID).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
	})
}
