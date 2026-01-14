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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendForgetPasswordMessageHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, requestBody io.Reader) *http.Request {
		t.Helper()

		return httptest.NewRequest(http.MethodPost, "/forget-password/send", requestBody)
	}

	// Вспомогательная функция для создания SendForgetPasswordMessageDTO
	createForgetPasswordDTO := func() domains.SendForgetPasswordMessageDTO {
		return domains.SendForgetPasswordMessageDTO{
			Email: "user@example.com",
		}
	}

	t.Run("successful send forget password message", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(nil)

		req := createRequest(t, bytes.NewReader(requestBody))
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
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

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
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Невалидный JSON
		invalidJSON := `{"email": invalid}`
		req := createRequest(t, bytes.NewReader([]byte(invalidJSON)))
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
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// JSON без обязательного поля
		missingFieldJSON := `{}`

		var dto domains.SendForgetPasswordMessageDTO

		err := json.Unmarshal([]byte(missingFieldJSON), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.Email) // Поле будет пустым

		// Use case может вернуть ошибку валидации или UserNotFound
		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), "").
			Return(customerrors.ErrUserNotFound) // Пустой email = пользователь не найден

		req := createRequest(t, bytes.NewReader([]byte(missingFieldJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - empty email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := domains.SendForgetPasswordMessageDTO{
			Email: "", // Пустой email
		}
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), "").
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("forbidden - email not confirmed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(customerrors.ErrEmailNotConfirmed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrEmailNotConfirmed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(errors.New("email service unavailable"))

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "email service unavailable")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("different email formats", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			email       string
			expectError bool
			errorType   error
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
				name:        "invalid email format",
				email:       "not-an-email",
				expectError: true,
				errorType:   customerrors.ErrUserNotFound, // Невалидный email = пользователь не найден
			},
			{
				name:        "email with spaces",
				email:       "user @example.com",
				expectError: true,
				errorType:   customerrors.ErrUserNotFound,
			},
			{
				name:        "case insensitive email",
				email:       "USER@EXAMPLE.COM",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

				dto := domains.SendForgetPasswordMessageDTO{
					Email: tc.email,
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				if tc.expectError {
					mockUseCase.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), tc.email).
						Return(tc.errorType)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					switch {
					case errors.Is(tc.errorType, customerrors.ErrUserNotFound):
						assert.Equal(t, http.StatusNotFound, rr.Code)
					case errors.Is(tc.errorType, customerrors.ErrEmailNotConfirmed):
						assert.Equal(t, http.StatusForbidden, rr.Code)
					}

					assert.Contains(t, rr.Body.String(), tc.errorType.Error())
				} else {
					mockUseCase.EXPECT().
						SendForgetPasswordMessage(gomock.Any(), tc.email).
						Return(nil)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusNoContent, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		const numRequests = 5

		// Каждый запрос с разными email
		emails := []string{
			"user1@example.com",
			"user2@example.com",
			"user3@example.com",
			"user4@example.com",
			"user5@example.com",
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			mockUseCase.EXPECT().
				SendForgetPasswordMessage(gomock.Any(), emails[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				dto := domains.SendForgetPasswordMessageDTO{
					Email: emails[idx],
				}
				requestBody, _ := json.Marshal(dto)
				req := createRequest(t, bytes.NewReader(requestBody))
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
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"email": "user@example.com",
			"extraField": "should be ignored",
			"anotherExtra": 123,
			"username": "user123"
		}`

		expectedEmail := "user@example.com"

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), expectedEmail).
			Return(nil)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("null email in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// JSON с null значением email
		jsonWithNull := `{"email": null}`

		// При десериализации null в string станет пустой строкой
		var dto domains.SendForgetPasswordMessageDTO

		err := json.Unmarshal([]byte(jsonWithNull), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.Email)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), "").
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должна быть ошибка "пользователь не найден"
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
	})

	t.Run("rate limiting scenarios", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Многократные запросы для одного email
		dto := domains.SendForgetPasswordMessageDTO{
			Email: "user@example.com",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Несколько запросов подряд
		const attempts = 3
		for range attempts {
			mockUseCase.EXPECT().
				SendForgetPasswordMessage(gomock.Any(), dto.Email).
				Return(nil) // Всегда успешно (rate limiting на уровне use case)
		}

		// Act & Assert - несколько запросов
		for range attempts {
			req := createRequest(t, bytes.NewReader(requestBody))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusNoContent, rr.Code)
		}
	})

	t.Run("email with special characters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Email со специальными символами (обычно допустимыми в локальной части)
		dto := domains.SendForgetPasswordMessageDTO{
			Email: "user.name+tag@example-domain.co.uk",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("non-existent domain email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Email с несуществующим доменом
		dto := domains.SendForgetPasswordMessageDTO{
			Email: "user@nonexistent-domain-12345.com",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Use case может вернуть UserNotFound или успешно обработать
		// (проверка существования домена обычно не на этом уровне)
		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
	})

	t.Run("status code 204 No Content on success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - проверяем, что статус код 204, а не 200
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
		assert.Empty(t, rr.Body.String())
	})

	t.Run("case insensitive email matching", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Email в разном регистре
		testEmails := []string{
			"USER@EXAMPLE.COM",
			"User@Example.com",
			"user@example.com",
		}

		for _, email := range testEmails {
			dto := domains.SendForgetPasswordMessageDTO{
				Email: email,
			}

			requestBody, err := json.Marshal(dto)
			require.NoError(t, err)

			// В зависимости от реализации, email может нормализоваться
			mockUseCase.EXPECT().
				SendForgetPasswordMessage(gomock.Any(), email).
				Return(nil)

			req := createRequest(t, bytes.NewReader(requestBody))
			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, http.StatusNoContent, rr.Code)
		}
	})

	t.Run("email with international characters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

		// Email с международными символами (IDN)
		dto := domains.SendForgetPasswordMessageDTO{
			Email: "user@müller.de", // В Punycode: xn--mller-kva.de
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// В зависимости от реализации, email может нормализоваться в Punycode
		mockUseCase.EXPECT().
			SendForgetPasswordMessage(gomock.Any(), dto.Email).
			Return(nil) // или ErrUserNotFound, если не поддерживается

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - зависит от поддержки IDN
		assert.Equal(t, http.StatusNoContent, rr.Code)
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
				name:       "email service error",
				error:      errors.New("SMTP server unavailable"),
				statusCode: http.StatusInternalServerError,
			},
			{
				name:       "rate limit exceeded in service",
				error:      errors.New("rate limit exceeded"),
				statusCode: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := auth.SendForgetPasswordMessageHandler(mockUseCase)

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				mockUseCase.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), dto.Email).
					Return(tc.error)

				req := createRequest(t, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, tc.statusCode, rr.Code)
				assert.Contains(t, rr.Body.String(), tc.error.Error())
			})
		}
	})
}
