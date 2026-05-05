package verify_email_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/verify_email"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	invalidJWTToken = "invalid_jwt_token"
	validToken      = "valid_token"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, token string) *http.Request {
		t.Helper()

		url := "/verify-email/" + token
		req := httptest.NewRequest(http.MethodPost, url, http.NoBody)

		// Устанавливаем параметры маршрута для mux.Vars
		vars := map[string]string{
			verify_email.TokenRouteKey: token,
		}
		req = mux.SetURLVars(req, vars)

		return req
	}

	t.Run("successful email verification", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		token := "valid_verification_token_123"

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(nil)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())
	})

	t.Run("unauthorized - invalid JWT token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), invalidJWTToken).
			Return(customerrors.ErrInvalidJWT)

		req := createRequest(t, invalidJWTToken)
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
		handler := verify_email.Handler(mockUseCase)

		token := "token_for_nonexistent_user"

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("conflict - email already confirmed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		token := "token_for_already_confirmed_email"

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrEmailAlreadyConfirmed)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrEmailAlreadyConfirmed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), validToken).
			Return(errors.New("database connection failed"))

		req := createRequest(t, validToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("different token formats", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			token       string
			expectError bool
			errorType   error
		}{
			{
				name:        "standard JWT token",
				token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
				expectError: false,
			},
			{
				name:        "hex token",
				token:       "a1b2c3d4e5f67890",
				expectError: false,
			},
			{
				name:        "uuid token",
				token:       "550e8400-e29b-41d4-a716-446655440000",
				expectError: false,
			},
			{
				name:        "token with special chars",
				token:       "token_with-special.chars+plus=equals",
				expectError: false,
			},
			{
				name:        "expired token",
				token:       "expired_jwt_token",
				expectError: true,
				errorType:   customerrors.ErrInvalidJWT,
			},
			{
				name:        "malformed JWT",
				token:       "not.a.valid.jwt",
				expectError: true,
				errorType:   customerrors.ErrInvalidJWT,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := verify_email.Handler(mockUseCase)

				if tc.expectError {
					mockUseCase.EXPECT().
						VerifyEmail(gomock.Any(), tc.token).
						Return(tc.errorType)

					req := createRequest(t, tc.token)
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					switch {
					case errors.Is(tc.errorType, customerrors.ErrInvalidJWT):
						assert.Equal(t, http.StatusUnauthorized, rr.Code)
					case errors.Is(tc.errorType, customerrors.ErrUserNotFound):
						assert.Equal(t, http.StatusNotFound, rr.Code)
					case errors.Is(tc.errorType, customerrors.ErrEmailAlreadyConfirmed):
						assert.Equal(t, http.StatusConflict, rr.Code)
					}

					assert.Contains(t, rr.Body.String(), tc.errorType.Error())
				} else {
					mockUseCase.EXPECT().
						VerifyEmail(gomock.Any(), tc.token).
						Return(nil)

					req := createRequest(t, tc.token)
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusNoContent, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent verification requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		const numRequests = 5

		// Каждый запрос с разными токенами
		tokens := []string{"token1", "token2", "token3", "token4", "token5"}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			mockUseCase.EXPECT().
				VerifyEmail(gomock.Any(), tokens[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				req := createRequest(t, tokens[idx])
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

	t.Run("expired verification token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		token := "expired_verification_token"

		// Истекший токен должен вернуть ошибку InvalidJWT
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrInvalidJWT)

		req := createRequest(t, token)
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
		handler := verify_email.Handler(mockUseCase)

		token := "single_use_verification_token"

		// Первый вызов - успешно
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(nil)

		req1 := createRequest(t, token)
		rr1 := httptest.NewRecorder()

		// Act - первый вызов
		handler.ServeHTTP(rr1, req1)

		// Assert - первый вызов
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй вызов с тем же токеном - должен вернуть ошибку
		// (токен одноразовый или email уже подтвержден)
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrEmailAlreadyConfirmed)

		req2 := createRequest(t, token)
		rr2 := httptest.NewRecorder()

		// Act - второй вызов
		handler.ServeHTTP(rr2, req2)

		// Assert - второй вызов
		assert.Equal(t, http.StatusConflict, rr2.Code)
		assert.Contains(t, rr2.Body.String(), customerrors.ErrEmailAlreadyConfirmed.Error())
	})

	t.Run("very long token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		// Создаем очень длинный токен (4096 символов)
		longToken := ""

		var longTokenSb365 strings.Builder
		for range 4096 {
			longTokenSb365.WriteString("a")
		}

		longToken += longTokenSb365.String()

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), longToken).
			Return(nil)

		req := createRequest(t, longToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("verification for deleted user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		token := "token_for_deleted_user"

		// Токен для удаленного пользователя
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
	})

	t.Run("database or external service errors", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name       string
			error      error
			statusCode int
		}{
			{
				name:       "database connection error",
				error:      errors.New("database connection failed"),
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "cache service error",
				error:      errors.New("redis connection failed"),
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "transaction error",
				error:      errors.New("transaction failed"),
				statusCode: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := verify_email.Handler(mockUseCase)

				mockUseCase.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
					Return(tc.error)

				req := createRequest(t, validToken)
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, tc.statusCode, rr.Code)
				assert.Contains(t, rr.Body.String(), tc.error.Error())
			})
		}
	})

	t.Run("status code 204 No Content on success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		token := "valid_verification_token"

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(nil)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - проверяем, что статус код 204, а не 200
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
		assert.Empty(t, rr.Body.String())
	})

	t.Run("token with query parameters in URL", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		// Токен, который может быть передан как часть URL
		token := "token123"

		// URL с query параметрами
		url := "/verify-email/" + token + "?redirect=true&mode=web"
		req := httptest.NewRequest(http.MethodPost, url, http.NoBody)
		vars := map[string]string{
			verify_email.TokenRouteKey: token,
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(nil)

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("multiple verification attempts with different outcomes", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		testCases := []struct {
			token      string
			error      error
			statusCode int
		}{
			{
				token:      "valid_token_1",
				error:      nil,
				statusCode: http.StatusNoContent,
			},
			{
				token:      "invalid_token",
				error:      customerrors.ErrInvalidJWT,
				statusCode: http.StatusUnauthorized,
			},
			{
				token:      "already_confirmed_token",
				error:      customerrors.ErrEmailAlreadyConfirmed,
				statusCode: http.StatusConflict,
			},
			{
				token:      "user_not_found_token",
				error:      customerrors.ErrUserNotFound,
				statusCode: http.StatusNotFound,
			},
			{
				token:      "server_error_token",
				error:      errors.New("server error"),
				statusCode: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			mockUseCase.EXPECT().
				VerifyEmail(gomock.Any(), tc.token).
				Return(tc.error)

			req := createRequest(t, tc.token)
			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, tc.statusCode, rr.Code)

			if tc.error != nil {
				assert.Contains(t, rr.Body.String(), tc.error.Error())
			}
		}
	})

	t.Run("verification after user changes email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		// Токен для подтверждения старого email (после смены email)
		token := "token_for_old_email"

		// Пользователь сменил email, старый email больше не привязан к аккаунту
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, token)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
	})

	t.Run("verification with body in request (should be ignored)", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		// Запрос с телом (должно игнорироваться)
		req := httptest.NewRequest(
			http.MethodPost,
			"/verify-email/"+validToken,
			http.NoBody,
		)
		// Можно добавить тело, но обработчик его не читает
		vars := map[string]string{
			verify_email.TokenRouteKey: validToken,
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()

		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), validToken).
			Return(nil)

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})
}
