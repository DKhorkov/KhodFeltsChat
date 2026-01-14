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
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestChangePasswordHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64, requestBody io.Reader) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/change-password", requestBody)
		ctx := contextlib.WithValue(req.Context(), middlewares.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания ChangePasswordDTO
	createChangePasswordDTO := func() domains.ChangePasswordDTO {
		return domains.ChangePasswordDTO{
			OldPassword: "OldSecurePassword123!",
			NewPassword: "NewSecurePassword123!",
		}
	}

	t.Run("successful password change", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Проверяем, что DTO получает правильный UserID
		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Запрос БЕЗ userID в контексте
		req := httptest.NewRequest(
			http.MethodPost,
			"/change-password",
			bytes.NewReader(requestBody),
		)
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
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Запрос с userID неверного типа в контексте
		req := httptest.NewRequest(
			http.MethodPost,
			"/change-password",
			bytes.NewReader(requestBody),
		)
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

	t.Run("bad request - empty request body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

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
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)

		// Невалидный JSON
		invalidJSON := `{"oldPassword": "old", "newPassword": invalid}`
		req := createRequest(t, userID, bytes.NewReader([]byte(invalidJSON)))
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
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)

		testCases := []struct {
			name string
			json string
		}{
			{
				name: "missing oldPassword",
				json: `{"newPassword": "NewPassword123!"}`,
			},
			{
				name: "missing newPassword",
				json: `{"oldPassword": "OldPassword123!"}`,
			},
			{
				name: "empty object",
				json: `{}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// DTO будет создан с пустыми полями
				var dto domains.ChangePasswordDTO

				err := json.Unmarshal([]byte(tc.json), &dto)
				require.NoError(t, err, "JSON should unmarshal even with missing fields")

				dto.UserID = userID

				mockUseCase.EXPECT().
					ChangePassword(gomock.Any(), dto).
					Return(customerrors.ErrValidationFailed)

				req := createRequest(t, userID, bytes.NewReader([]byte(tc.json)))
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
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := domains.ChangePasswordDTO{
			OldPassword: "old", // Слишком короткий старый пароль
			NewPassword: "123", // Слишком короткий новый пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - wrong old password", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(customerrors.ErrWrongPassword)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(999) // Несуществующий пользователь
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(customerrors.ErrUserNotFound)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
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
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(errors.New("database connection failed"))

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("different password strengths for new password", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			newPassword string
			expectError bool
		}{
			{
				name:        "strong password",
				newPassword: "NewSecurePassword123!",
				expectError: false,
			},
			{
				name:        "very strong password",
				newPassword: "V3ry$tr0ngN3wP@ssw0rd!2024",
				expectError: false,
			},
			{
				name:        "weak password",
				newPassword: "123",
				expectError: true,
			},
			{
				name:        "common password",
				newPassword: "password",
				expectError: true,
			},
			{
				name:        "same as old password",
				newPassword: "OldPassword123!", // Предполагаем, что такой же как старый
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := auth.ChangePasswordHandler(mockUseCase)

				userID := uint64(123)
				dto := domains.ChangePasswordDTO{
					OldPassword: "OldPassword123!",
					NewPassword: tc.newPassword,
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				expectedDTO := dto
				expectedDTO.UserID = userID

				if tc.expectError {
					mockUseCase.EXPECT().
						ChangePassword(gomock.Any(), expectedDTO).
						Return(customerrors.ErrValidationFailed)

					req := createRequest(t, userID, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusBadRequest, rr.Code)
				} else {
					mockUseCase.EXPECT().
						ChangePassword(gomock.Any(), expectedDTO).
						Return(nil)

					req := createRequest(t, userID, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusNoContent, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent password change requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		const numRequests = 5

		// Каждый запрос от разных пользователей
		userIDs := []uint64{1, 2, 3, 4, 5}
		oldPasswords := []string{"Pass1!", "Pass2!", "Pass3!", "Pass4!", "Pass5!"}
		newPasswords := []string{"New1!", "New2!", "New3!", "New4!", "New5!"}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedDTO := domains.ChangePasswordDTO{
				UserID:      userIDs[i],
				OldPassword: oldPasswords[i],
				NewPassword: newPasswords[i],
			}

			mockUseCase.EXPECT().
				ChangePassword(gomock.Any(), expectedDTO).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				dto := domains.ChangePasswordDTO{
					OldPassword: oldPasswords[idx],
					NewPassword: newPasswords[idx],
				}
				requestBody, _ := json.Marshal(dto)
				req := createRequest(t, userIDs[idx], bytes.NewReader(requestBody))
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
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"oldPassword": "OldPassword123!",
			"newPassword": "NewPassword123!",
			"extraField": "should be ignored",
			"anotherExtra": 123,
			"confirmPassword": "NewPassword123!"
		}`

		expectedDTO := domains.ChangePasswordDTO{
			UserID:      userID,
			OldPassword: "OldPassword123!",
			NewPassword: "NewPassword123!",
		}

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(nil)

		req := createRequest(t, userID, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("null values in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)

		// JSON с null значениями
		jsonWithNull := `{
			"oldPassword": null,
			"newPassword": null
		}`

		// При десериализации null в string станет пустой строкой
		var dto domains.ChangePasswordDTO

		err := json.Unmarshal([]byte(jsonWithNull), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.OldPassword)
		assert.Equal(t, "", dto.NewPassword)

		dto.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), dto).
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, userID, bytes.NewReader([]byte(jsonWithNull)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должна быть ошибка валидации
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("empty strings for passwords", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := domains.ChangePasswordDTO{
			OldPassword: "", // Пустой старый пароль
			NewPassword: "", // Пустой новый пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(customerrors.ErrValidationFailed)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("zero user ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(0) // Нулевой ID
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(nil) // В зависимости от бизнес-логики

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться (если userID=0 допустим)
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("very large user ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(18446744073709551615) // Максимальный uint64
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("multiple password change attempts", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Ожидаем два вызова ChangePassword для одного пользователя
		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Times(2).
			Return(nil)

		// Первый запрос
		req1 := createRequest(t, userID, bytes.NewReader(requestBody))
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)

		// Assert - первый запрос
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй запрос (повторная смена пароля)
		req2 := createRequest(t, userID, bytes.NewReader(requestBody))
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

		// Assert - второй запрос
		assert.Equal(t, http.StatusNoContent, rr2.Code)
	})

	t.Run("status code 204 No Content on success", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(nil)

		req := createRequest(t, userID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - проверяем, что статус код 204, а не 200
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
		assert.Empty(t, rr.Body.String())
	})

	t.Run("user changes password for another user (should not be possible)", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := auth.ChangePasswordHandler(mockUseCase)

		// Пользователь пытается изменить пароль другому пользователю
		authenticatedUserID := uint64(123) // ID аутентифицированного пользователя
		otherUserID := uint64(456)         // ID другого пользователя

		// DTO с UserID другого пользователя (должен быть перезаписан)
		dto := domains.ChangePasswordDTO{
			UserID:      otherUserID, // Другой пользователь
			OldPassword: "OldPassword123!",
			NewPassword: "NewPassword123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Ожидаем, что UserID в DTO будет перезаписан аутентифицированным userID
		expectedDTO := dto
		expectedDTO.UserID = authenticatedUserID // Должен быть перезаписан!

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Return(customerrors.ErrWrongPassword) // Потому что старый пароль не совпадает

		req := createRequest(t, authenticatedUserID, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - пользователь не может изменить пароль другому пользователю
		// так как UserID перезаписывается из контекста
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
	})
}
