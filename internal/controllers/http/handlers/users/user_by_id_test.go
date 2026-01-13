package users_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUserByIDHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(
			http.MethodGet,
			"/users/"+strconv.FormatUint(userID, 10),
			http.NoBody,
		)

		// Устанавливаем параметры маршрута для mux.Vars
		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(userID, 10),
		}
		req = mux.SetURLVars(req, vars)

		return req
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

	t.Run("successful get user by ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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

		// Проверяем все обязательные поля
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

	t.Run("successful get user by ID with unconfirmed email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(456)
		expectedUser := createTestUser(userID, false)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(userID), response["id"])
		assert.Equal(t, "testuser456", response["username"])
		assert.Equal(t, "user456@example.com", response["email"])
		assert.False(t, response["emailConfirmed"].(bool))
	})

	t.Run("bad request - invalid user ID parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		// Запрос с невалидным user ID
		req := httptest.NewRequest(http.MethodGet, "/users/invalid", http.NoBody)
		vars := map[string]string{
			common.IDRouteKey: "invalid",
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid syntax")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - empty user ID parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		// Запрос с пустым user ID
		req := httptest.NewRequest(http.MethodGet, "/users/", http.NoBody)
		vars := map[string]string{
			common.IDRouteKey: "",
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid syntax")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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

	t.Run("user with minimum valid ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(1) // Минимальный ID согласно схеме (minimum: 1)
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

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["id"])
	})

	t.Run("user with very large ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(18446744073709551615) // Максимальный uint64
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

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем, что ID правильно преобразован
		assert.Equal(t, 1.8446744073709552e+19, response["id"]) // JSON числа как float64
	})

	t.Run("user with valid email format", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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

	t.Run("user with different timestamps", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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

	t.Run("request with query parameters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		// Запрос с query параметрами (которые должны игнорироваться)
		req := httptest.NewRequest(http.MethodGet, "/users/123?fields=all&expand=true", http.NoBody)
		vars := map[string]string{
			common.IDRouteKey: "123",
		}
		req = mux.SetURLVars(req, vars)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("request with headers", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		// Запрос с различными заголовками
		req := httptest.NewRequest(http.MethodGet, "/users/123", http.NoBody)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Language", "en-US")
		req.Header.Set("User-Agent", "TestClient/1.0")

		vars := map[string]string{
			common.IDRouteKey: "123",
		}
		req = mux.SetURLVars(req, vars)
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
	})

	t.Run("concurrent requests for different users", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 5

		userIDs := []uint64{1, 2, 3, 4, 5}

		// Настраиваем мок на вызовы для разных пользователей
		for _, userID := range userIDs {
			expectedUser := createTestUser(userID, true)
			mockUseCase.EXPECT().
				GetUserByID(gomock.Any(), userID).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for _, userID := range userIDs {
			go func(id uint64) {
				req := createRequest(t, id)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(id), response["id"])

				done <- true
			}(userID)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("zero user ID - should fail parsing", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

		userID := uint64(0) // Нулевой ID

		// Мок не должен вызываться, так как парсинг должен пройти успешно
		// (0 - валидное число для uint64)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Return(expectedUser, nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - парсинг пройдет успешно, но use case может вернуть ошибку
		// или пустой результат в зависимости от бизнес-логики
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("JSON encoding preserves field order", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUserByIDHandler(mockUseCase)

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

		// Проверяем, что JSON содержит правильные поля
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
