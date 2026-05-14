package update_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/settings/update"
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

	createRequest := func(t *testing.T, userID uint64, body []byte) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPut, "/api/users/me/settings", bytes.NewReader(body))
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	t.Run("successful update settings to dark theme", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		mockUseCase.EXPECT().
			UpdateSettings(gomock.Any(), domains.Settings{UserID: userID, Theme: domains.ThemeDark}).
			Return(&domains.Settings{
				ID:        1,
				UserID:    userID,
				Theme:     domains.ThemeDark,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil)

		body, _ := json.Marshal(map[string]any{"theme": 1})
		req := createRequest(t, userID, body)
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

		assert.Equal(t, float64(1), response["theme"])
	})

	t.Run("successful update settings to light theme", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(456)
		now := time.Now()

		mockUseCase.EXPECT().
			UpdateSettings(gomock.Any(), domains.Settings{UserID: userID, Theme: domains.ThemeLight}).
			Return(&domains.Settings{
				ID:        2,
				UserID:    userID,
				Theme:     domains.ThemeLight,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil)

		body, _ := json.Marshal(map[string]any{"theme": 0})
		req := createRequest(t, userID, body)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(0), response["theme"])
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		body, _ := json.Marshal(map[string]any{"theme": 1})
		req := httptest.NewRequest(http.MethodPut, "/api/users/me/settings", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		req := createRequest(t, userID, []byte("invalid json"))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("settings not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(999)

		mockUseCase.EXPECT().
			UpdateSettings(gomock.Any(), gomock.Any()).
			Return(nil, customerrors.ErrSettingsNotFound)

		body, _ := json.Marshal(map[string]any{"theme": 1})
		req := createRequest(t, userID, body)
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
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			UpdateSettings(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("database error"))

		body, _ := json.Marshal(map[string]any{"theme": 1})
		req := createRequest(t, userID, body)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database error")
	})

	t.Run("userID from context is used, not from body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		// Тело содержит другой userID, но хэндлер должен использовать userID из контекста
		mockUseCase.EXPECT().
			UpdateSettings(gomock.Any(), domains.Settings{UserID: userID, Theme: domains.ThemeDark}).
			Return(&domains.Settings{
				ID:        1,
				UserID:    userID,
				Theme:     domains.ThemeDark,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil)

		body, _ := json.Marshal(map[string]any{"theme": 1, "userID": 999})
		req := createRequest(t, userID, body)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
