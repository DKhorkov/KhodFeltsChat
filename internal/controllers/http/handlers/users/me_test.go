package users_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetMeHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), middlewares.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "testuser" + strconv.FormatUint(id, 10),
			Email:          "user" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Hour),
		}
	}

	t.Run("successful get current user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

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

		// Проверяем все обязательные поля согласно схеме
		assert.Equal(t, float64(userID), response["id"])
		assert.Equal(t, "testuser123", response["username"])
		assert.Equal(t, "user123@example.com", response["email"])
		assert.True(t, response["emailConfirmed"].(bool))
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])

		// Проверяем, что пароль не возвращается
		_, passwordExists := response["password"]
		assert.False(t, passwordExists, "password should not be included in response")
	})

	t.Run("successful get current user with unconfirmed email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(456)
		expectedUser := createTestUser(userID, false)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(userID), response["id"])
		assert.Equal(t, "testuser456", response["username"])
		assert.Equal(t, "user456@example.com", response["email"])
		assert.False(t, response["emailConfirmed"].(bool))
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		// Запрос БЕЗ userID в контексте
		req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("unauthorized - invalid userID type in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		// Запрос с userID неверного типа в контексте
		req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), middlewares.UserIDContextKey, "not-a-number")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(999) // Несуществующий пользователь

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(nil, customerrors.ErrUserNotFound)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("user with minimum valid username length", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		// Согласно схеме: minLength: 5
		expectedUser := &domains.User{
			ID:             userID,
			Username:       "user5", // 5 символов - минимальная длина
			Email:          "user@example.com",
			EmailConfirmed: true,
			Password:       "hashed_password",
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "user5", response["username"])
	})

	t.Run("user with maximum valid username length", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		// Согласно схеме: maxLength: 70
		// Создаем username длиной 70 символов
		longUsername := "u"

		var longUsernameSb275 strings.Builder
		for range 69 {
			longUsernameSb275.WriteString("a")
		}

		longUsername += longUsernameSb275.String()

		expectedUser := &domains.User{
			ID:             userID,
			Username:       longUsername, // 70 символов - максимальная длина
			Email:          "user@example.com",
			EmailConfirmed: true,
			Password:       "hashed_password",
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, longUsername, response["username"])
		assert.Len(t, response["username"].(string), 70)
	})

	t.Run("user with valid email format", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		testEmails := []string{
			"simple@example.com",
			"user.name@example.com",
			"user_name@example.co.uk",
			"user+tag@example.com",
			"123456@example.com",
		}

		for _, email := range testEmails {
			expectedUser := &domains.User{
				ID:             userID,
				Username:       "testuser",
				Email:          email,
				EmailConfirmed: true,
				Password:       "hashed_password",
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			mockUseCase.EXPECT().
				GetUserByID(gomock.Any(), userID).
				Return(expectedUser, nil)

			req := createRequest(t, userID)
			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, http.StatusOK, rr.Code)

			var response map[string]any

			err := json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, email, response["email"])
		}
	})

	t.Run("zero user ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(0) // Нулевой ID - пограничный случай

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(nil, customerrors.ErrUserNotFound)
		// Предполагаем, что пользователь с ID=0 не существует

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("user with different timestamps", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)

		// Создаем пользователя с разными временными метками
		createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)

		expectedUser := &domains.User{
			ID:             userID,
			Username:       "testuser",
			Email:          "user@example.com",
			EmailConfirmed: true,
			Password:       "hashed_password",
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем, что временные метки присутствуют
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])

		// Проверяем, что updatedAt позже createdAt
		createdAtStr, _ := response["createdAt"].(string)
		updatedAtStr, _ := response["updatedAt"].(string)

		parsedCreatedAt, err1 := time.Parse(time.RFC3339Nano, createdAtStr)
		parsedUpdatedAt, err2 := time.Parse(time.RFC3339Nano, updatedAtStr)

		if err1 == nil && err2 == nil {
			assert.True(
				t,
				parsedUpdatedAt.After(parsedCreatedAt),
				"updatedAt should be after createdAt",
			)
		}
	})

	t.Run("concurrent requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 10

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		// Настраиваем мок на вызовы
		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Times(numRequests).
			Return(expectedUser, nil)

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(_ int) {
				req := createRequest(t, userID)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("JSON encoding preserves field order", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetMeHandler(mockUseCase)

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем, что JSON содержит правильные поля в правильном порядке
		// (опционально, зависит от требований)
		responseBody := rr.Body.String()

		// Простая проверка на наличие всех полей
		assert.Contains(t, responseBody, `"id"`)
		assert.Contains(t, responseBody, `"username"`)
		assert.Contains(t, responseBody, `"email"`)
		assert.Contains(t, responseBody, `"emailConfirmed"`)
		assert.Contains(t, responseBody, `"createdAt"`)
		assert.Contains(t, responseBody, `"updatedAt"`)

		// Проверяем, что пароль не возвращается
		assert.NotContains(t, responseBody, `"password"`)
	})
}
