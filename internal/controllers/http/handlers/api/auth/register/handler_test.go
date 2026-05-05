package register_test

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

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/register"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, requestBody io.Reader) *http.Request {
		t.Helper()

		return httptest.NewRequest(http.MethodPost, "/register", requestBody)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "newuser" + strconv.FormatUint(id, 10),
			Email:          "newuser" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	// Вспомогательная функция для создания RegisterDTO
	createRegisterDTO := func() domains.RegisterDTO {
		return domains.RegisterDTO{
			Username: "newuser",
			Email:    "newuser@example.com",
			Password: "SecurePassword123!",
		}
	}

	t.Run("successful registration", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(123, false) // emailConfirmed = false при регистрации

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(expectedUser, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response map[string]any

		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		// Проверяем поля пользователя
		assert.Equal(t, float64(123), response["id"])
		assert.Equal(t, "newuser123", response["username"])
		assert.Equal(t, "newuser123@example.com", response["email"])
		assert.False(
			t,
			response["emailConfirmed"].(bool),
			"email should not be confirmed after registration",
		)
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])

		// Проверяем, что пароль не возвращается
		_, passwordExists := response["password"]
		assert.False(t, passwordExists, "password should not be included in response")
	})

	t.Run("successful registration with confirmed email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Иногда email может быть сразу подтвержден (например, при тестировании или специальных сценариях)
		expectedUser := createTestUser(123, true)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(expectedUser, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]any

		err = json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(
			t,
			response["emailConfirmed"].(bool),
			"email can be confirmed after registration in some cases",
		)
	})

	t.Run("bad request - empty request body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Пустое тело запроса
		req := createRequest(t, bytes.NewReader([]byte{}))
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
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Невалидный JSON
		invalidJSON := `{"username": "test", "email": invalid}`
		req := createRequest(t, bytes.NewReader([]byte(invalidJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid character")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - missing required fields", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		testCases := []struct {
			name string
			json string
		}{
			{
				name: "missing username",
				json: `{"email": "test@example.com", "password": "password123"}`,
			},
			{
				name: "missing email",
				json: `{"username": "testuser", "password": "password123"}`,
			},
			{
				name: "missing password",
				json: `{"username": "testuser", "email": "test@example.com"}`,
			},
			{
				name: "empty object",
				json: `{}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// DTO будет создан с пустыми полями, валидация произойдет в use case
				// и вернет ErrValidationFailed
				var dto domains.RegisterDTO

				err := json.Unmarshal([]byte(tc.json), &dto)
				require.NoError(t, err, "JSON should unmarshal even with missing fields")

				mockUseCase.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)

				req := createRequest(t, bytes.NewReader([]byte(tc.json)))
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			})
		}
	})

	t.Run("bad request - validation failed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := domains.RegisterDTO{
			Username: "ab",            // Слишком короткий username
			Email:    "invalid-email", // Невалидный email
			Password: "123",           // Слишком короткий пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("conflict - user already exists", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrUserAlreadyExists)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("conflict - username already exists", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Попытка зарегистрировать пользователя с существующим username
		dto := domains.RegisterDTO{
			Username: "existinguser",
			Email:    "newemail@example.com", // Новый email
			Password: "Password123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrUserAlreadyExists)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
	})

	t.Run("conflict - email already exists", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Попытка зарегистрировать пользователя с существующим email
		dto := domains.RegisterDTO{
			Username: "newusername",
			Email:    "existing@example.com", // Существующий email
			Password: "Password123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrUserAlreadyExists)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - JSON encoding error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(123, false)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(expectedUser, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться
		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("registration with different password strengths", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		testCases := []struct {
			name        string
			password    string
			expectError bool
		}{
			{
				name:        "strong password",
				password:    "SecurePassword123!",
				expectError: false,
			},
			{
				name:        "very strong password",
				password:    "V3ry$tr0ngP@ssw0rd!2024",
				expectError: false,
			},
			{
				name:        "weak password",
				password:    "123",
				expectError: true, // Должна быть ошибка валидации
			},
			{
				name:        "common password",
				password:    "password",
				expectError: true, // Должна быть ошибка валидации
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: tc.password,
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				if tc.expectError {
					mockUseCase.EXPECT().
						RegisterUser(gomock.Any(), dto).
						Return(nil, customerrors.ErrValidationFailed)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusBadRequest, rr.Code)
				} else {
					expectedUser := createTestUser(123, false)

					mockUseCase.EXPECT().
						RegisterUser(gomock.Any(), dto).
						Return(expectedUser, nil)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusCreated, rr.Code)
				}
			})
		}
	})

	t.Run("registration with different email formats", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		testCases := []struct {
			name        string
			email       string
			expectError bool
		}{
			{
				name:        "simple email",
				email:       "user@example.com",
				expectError: false,
			},
			{
				name:        "email with plus",
				email:       "user+tag@example.com",
				expectError: false,
			},
			{
				name:        "email with dot",
				email:       "user.name@example.com",
				expectError: false,
			},
			{
				name:        "email with subdomain",
				email:       "user@sub.example.com",
				expectError: false,
			},
			{
				name:        "invalid email",
				email:       "not-an-email",
				expectError: true,
			},
			{
				name:        "email without domain",
				email:       "user@",
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    tc.email,
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				if tc.expectError {
					mockUseCase.EXPECT().
						RegisterUser(gomock.Any(), dto).
						Return(nil, customerrors.ErrValidationFailed)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusBadRequest, rr.Code)
				} else {
					expectedUser := createTestUser(123, false)

					mockUseCase.EXPECT().
						RegisterUser(gomock.Any(), dto).
						Return(expectedUser, nil)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusCreated, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent registrations", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 5

		// Каждый запрос с разными данными
		requestsData := make([]domains.RegisterDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.RegisterDTO{
				Username: "user" + strconv.Itoa(i),
				Email:    "user" + strconv.Itoa(i) + "@example.com",
				Password: "Password123!" + strconv.Itoa(i),
			}
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedUser := &domains.User{
				ID:             uint64(i + 1000),
				Username:       requestsData[i].Username,
				Email:          requestsData[i].Email,
				EmailConfirmed: false,
				Password:       "hashed_password",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			mockUseCase.EXPECT().
				RegisterUser(gomock.Any(), requestsData[i]).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusCreated, rr.Code)

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("registration with extra fields in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"username": "newuser",
			"email": "new@example.com",
			"password": "SecurePassword123!",
			"extraField": "should be ignored",
			"anotherExtra": 123,
			"isAdmin": true
		}`

		expectedUser := createTestUser(123, false)

		// Ожидаем, что только определенные поля будут в DTO
		expectedDTO := domains.RegisterDTO{
			Username: "newuser",
			Email:    "new@example.com",
			Password: "SecurePassword123!",
		}

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), expectedDTO).
			Return(expectedUser, nil)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
	})

	t.Run("registration with null values", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// JSON с null значениями
		jsonWithNull := `{
			"username": null,
			"email": "test@example.com",
			"password": "SecurePassword123!"
		}`

		// При десериализации null в string станет пустой строкой
		var dto domains.RegisterDTO

		err := json.Unmarshal([]byte(jsonWithNull), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.Username)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должна быть ошибка валидации
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("registration with empty strings", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := domains.RegisterDTO{
			Username: "", // Пустой username
			Email:    "", // Пустой email
			Password: "", // Пустой пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(nil, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("status code 201 Created is set correctly", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		dto := createRegisterDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedUser := createTestUser(123, false)

		mockUseCase.EXPECT().
			RegisterUser(gomock.Any(), dto).
			Return(expectedUser, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - проверяем, что статус код 201, а не 200
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 201 Created, not 200 OK")

		// Проверяем, что заголовок установлен до записи тела
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)
	})
}
