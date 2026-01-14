package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	missingFieldJSON = `{}`
)

func TestForgetPasswordHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, token string, requestBody io.Reader) *http.Request {
		t.Helper()

		url := "/forget-password/" + token
		req := httptest.NewRequest(http.MethodPost, url, requestBody)

		// Устанавливаем параметры маршрута для mux.Vars
		vars := map[string]string{
			auth.ForgetPasswordTokenRouteKey: token,
		}
		req = mux.SetURLVars(req, vars)

		return req
	}

	// Вспомогательная функция для создания ForgetPasswordDTO
	createForgetPasswordDTO := func() domains.ForgetPasswordDTO {
		return domains.ForgetPasswordDTO{
			NewPassword: "NewSecurePassword123!",
		}
	}

	t.Run("successful password reset", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "valid_reset_token_123"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(nil)

		req := createRequest(t, token, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())
	})

	t.Run("bad request - empty request body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		// Пустое тело запроса
		req := createRequest(t, validToken, bytes.NewReader([]byte{}))
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
		handler := auth.ForgetPasswordHandler(mockUseCase)

		// Невалидный JSON
		invalidJSON := `{"newPassword": invalid}`
		req := createRequest(t, validToken, bytes.NewReader([]byte(invalidJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid character")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - missing required field", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		var dto domains.ForgetPasswordDTO

		err := json.Unmarshal([]byte(missingFieldJSON), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.NewPassword) // Поле будет пустым

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, "").
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, validToken, bytes.NewReader([]byte(missingFieldJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - validation failed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		dto := domains.ForgetPasswordDTO{
			NewPassword: "123", // Слишком короткий пароль
		}
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, dto.NewPassword).
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, validToken, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("unauthorized - invalid JWT token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "invalid_jwt_token"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrInvalidJWT)

		req := createRequest(t, token, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "valid_token_for_nonexistent_user"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, token, bytes.NewReader(requestBody))
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
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, dto.NewPassword).
			Return(errors.New("database connection failed"))

		req := createRequest(t, validToken, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("different password strengths", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			password    string
			expectError bool
		}{
			{
				name:        "strong password",
				password:    "NewSecurePassword123!",
				expectError: false,
			},
			{
				name:        "very strong password",
				password:    "V3ry$tr0ngN3wP@ssw0rd!2024",
				expectError: false,
			},
			{
				name:        "weak password",
				password:    "123",
				expectError: true,
			},
			{
				name:        "common password",
				password:    "password",
				expectError: true,
			},
			{
				name:        "empty password",
				password:    "",
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := auth.ForgetPasswordHandler(mockUseCase)

				dto := domains.ForgetPasswordDTO{
					NewPassword: tc.password,
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				if tc.expectError {
					mockUseCase.EXPECT().
						ForgetPassword(gomock.Any(), validToken, tc.password).
						Return(customerrors.ErrValidationFailed)

					req := createRequest(t, validToken, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusBadRequest, rr.Code)
					assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				} else {
					mockUseCase.EXPECT().
						ForgetPassword(gomock.Any(), validToken, tc.password).
						Return(nil)

					req := createRequest(t, validToken, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusNoContent, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent password reset requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		const numRequests = 5

		// Каждый запрос с разными токенами
		tokens := []string{"token1", "token2", "token3", "token4", "token5"}
		passwords := []string{"Pass1!", "Pass2!", "Pass3!", "Pass4!", "Pass5!"}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			mockUseCase.EXPECT().
				ForgetPassword(gomock.Any(), tokens[i], passwords[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				dto := domains.ForgetPasswordDTO{
					NewPassword: passwords[idx],
				}
				requestBody, _ := json.Marshal(dto)
				req := createRequest(t, tokens[idx], bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("JSON with extra fields", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"newPassword": "NewSecurePassword123!",
			"extraField": "should be ignored",
			"anotherExtra": 123,
			"confirmPassword": "NewSecurePassword123!"
		}`

		expectedPassword := "NewSecurePassword123!"

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, expectedPassword).
			Return(nil)

		req := createRequest(t, validToken, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("null password in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		// JSON с null значением пароля
		jsonWithNull := `{"newPassword": null}`

		// При десериализации null в string станет пустой строкой
		var dto domains.ForgetPasswordDTO

		err := json.Unmarshal([]byte(jsonWithNull), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.NewPassword)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, "").
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, validToken, bytes.NewReader([]byte(jsonWithNull)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должна быть ошибка валидации
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("expired token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "expired_token"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Истекший токен должен вернуть ошибку InvalidJWT
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrInvalidJWT)

		req := createRequest(t, token, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
	})

	t.Run("token used multiple times", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "single_use_token"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Первый вызов - успешно
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(nil)

		req1 := createRequest(t, token, bytes.NewReader(requestBody))
		rr1 := httptest.NewRecorder()

		// Act - первый вызов
		handler.ServeHTTP(rr1, req1)

		// Assert - первый вызов
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй вызов с тем же токеном - должен вернуть ошибку
		// (токен одноразовый)
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrInvalidJWT)

		req2 := createRequest(t, token, bytes.NewReader(requestBody))
		rr2 := httptest.NewRecorder()

		// Act - второй вызов
		handler.ServeHTTP(rr2, req2)

		// Assert - второй вызов
		assert.Equal(t, http.StatusUnauthorized, rr2.Code)
		assert.Contains(t, rr2.Body.String(), customerrors.ErrInvalidJWT.Error())
	})

	t.Run("malformed JWT token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		token := "not.a.valid.jwt.token"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrInvalidJWT)

		req := createRequest(t, token, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
	})

	t.Run("same password as old password", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		// Пароль, который совпадает со старым паролем пользователя
		dto := domains.ForgetPasswordDTO{
			NewPassword: "SameAsOldPassword123!",
		}
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// В зависимости от бизнес-логики, это может быть ошибкой валидации
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, dto.NewPassword).
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, validToken, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("status code 204 No Content on success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ForgetPasswordHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), validToken, dto.NewPassword).
			Return(nil)

		req := createRequest(t, validToken, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - проверяем, что статус код 204, а не 200
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
		assert.Empty(t, rr.Body.String())
	})
}
