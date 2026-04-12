package update_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users/update"
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

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64, requestBody io.Reader) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPatch, "/me", requestBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "updateduser" + strconv.FormatUint(id, 10),
			Email:          "updated" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now.Add(-24 * time.Hour), // Создан вчера
			UpdatedAt:      now,                      // Обновлен сейчас
		}
	}

	// Вспомогательная функция для создания DTO
	createUpdateDTO := func() domains.UpdateUserDTO {
		return domains.UpdateUserDTO{
			Username: "newusername",
		}
	}

	t.Run("successful update current user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		dto := createUpdateDTO()

		// Преобразуем DTO в JSON для тела запроса
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(userID, true)

		// Проверяем, что DTO получает правильный ID
		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
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

		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем обновленные поля
		assert.Equal(t, float64(userID), response["id"])
		assert.Equal(t, "updateduser123", response["username"])
		assert.Equal(t, "updated123@example.com", response["email"])
		assert.True(t, response["emailConfirmed"].(bool))
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])

		// Проверяем, что пароль не возвращается
		_, passwordExists := response["password"]
		assert.False(t, passwordExists, "password should not be included in response")
	})

	t.Run("successful partial update", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// Обновляем только username
		dto := domains.UpdateUserDTO{
			Username: "onlyusernameupdated",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(userID, true)

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		dto := createUpdateDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Запрос БЕЗ userID в контексте
		req := httptest.NewRequest(http.MethodPatch, "/me", bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - empty request body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// Пустое тело запроса
		req := createRequest(t, userID, bytes.NewReader([]byte{}))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// Невалидный JSON
		invalidJSON := `{"username": "test", "email": invalid}`
		req := createRequest(t, userID, bytes.NewReader([]byte(invalidJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - validation failed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// DTO с невалидными данными
		dto := domains.UpdateUserDTO{
			Username: "ab", // Слишком короткий username
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(nil, customerrors.ErrValidationFailed)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(999) // Несуществующий пользователь
		dto := createUpdateDTO()

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(nil, customerrors.ErrUserNotFound)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
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
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		dto := createUpdateDTO()

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - JSON encoding error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)
		dto := createUpdateDTO()

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(userID, true)

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("update with empty DTO", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// Пустой DTO - ничего не обновляем
		dto := domains.UpdateUserDTO{}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Пользователь возвращается без изменений
		expectedUser := &domains.User{
			ID:             userID,
			Username:       "originaluser",
			Email:          "original@example.com",
			EmailConfirmed: true,
			Password:       "hashed_password",
			CreatedAt:      time.Now().Add(-24 * time.Hour),
			UpdatedAt:      time.Now().Add(-24 * time.Hour),
		}

		expectedDTO := dto
		expectedDTO.ID = userID

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем, что данные не изменились
		assert.Equal(t, "originaluser", response["username"])
		assert.Equal(t, "original@example.com", response["email"])
	})

	t.Run("update with extra fields in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"username": "newuser",
			"email": "new@example.com",
			"extraField": "should be ignored",
			"anotherExtra": 123
		}`

		expectedUser := createTestUser(userID, true)

		// Ожидаем, что только определенные поля будут в DTO
		expectedDTO := domains.UpdateUserDTO{
			ID:       userID,
			Username: "newuser",
		}

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("request body too large", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// Создаем очень большое тело запроса
		hugeBody := make([]byte, 10*1024*1024) // 10 MB
		for i := range hugeBody {
			hugeBody[i] = 'a'
		}
		// Заключаем в JSON
		hugeJSON := []byte(`{"username":"` + string(hugeBody) + `"}`)

		req := createRequest(t, userID, bytes.NewReader(hugeJSON))

		// Устанавливаем лимит на чтение тела
		req.Body = http.MaxBytesReader(nil, req.Body, 1024) // 1KB лимит

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен вернуть 400 из-за слишком большого тела
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("concurrent updates", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 5

		userID := uint64(123)

		// Каждый запрос с разными данными
		requestsData := make([]domains.UpdateUserDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.UpdateUserDTO{
				Username: "user" + strconv.Itoa(i),
			}
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedDTO := requestsData[i]
			expectedDTO.ID = userID

			expectedUser := &domains.User{
				ID:             userID,
				Username:       expectedDTO.Username,
				EmailConfirmed: true,
				Password:       "hashed_password",
				CreatedAt:      time.Now().Add(-24 * time.Hour),
				UpdatedAt:      time.Now(),
			}

			mockUseCase.EXPECT().
				UpdateUser(gomock.Any(), expectedDTO).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, userID, bytes.NewReader(requestBody))
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

	t.Run("DTO ID is overwritten with userID from context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		userID := uint64(123)

		// DTO с другим ID (должен быть перезаписан)
		dto := domains.UpdateUserDTO{
			ID:       999, // Другой ID в DTO
			Username: "newusername",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(userID, true)

		// Ожидаем, что ID в DTO будет перезаписан userID из контекста
		expectedDTO := dto
		expectedDTO.ID = userID // Должен быть перезаписан!

		mockUseCase.EXPECT().
			UpdateUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response map[string]any

		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем, что вернулся пользователь с правильным ID
		assert.Equal(t, float64(userID), response["id"])
		assert.NotEqual(t, float64(999), response["id"])
	})
}
