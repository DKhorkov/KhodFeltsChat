package change_password_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/change_password"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Вспомогательная функция для создания запроса
func createChangePasswordRequest(t *testing.T, userID uint64, requestBody io.Reader) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/change-password", requestBody)
	ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

	return req.WithContext(ctx)
}

// Вспомогательная функция для создания ChangePasswordDTO
func createChangePasswordDTO() domains.ChangePasswordDTO {
	return domains.ChangePasswordDTO{
		OldPassword: "OldSecurePassword123!",
		NewPassword: "NewSecurePassword123!",
	}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful password change",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(requestBody))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "unauthorized - invalid userID type in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/change-password", bytes.NewReader(requestBody))
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, "not-a-number")
				return req.WithContext(ctx)
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty request body",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte{}))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - invalid JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				invalidJSON := `{"oldPassword": "old", "newPassword": invalid}`
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(invalidJSON)))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid character")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - missing oldPassword",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(`{"newPassword": "NewPassword123!"}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				var dto domains.ChangePasswordDTO
				_ = json.Unmarshal([]byte(`{"newPassword": "NewPassword123!"}`), &dto)
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - missing newPassword",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(`{"oldPassword": "OldPassword123!"}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				var dto domains.ChangePasswordDTO
				_ = json.Unmarshal([]byte(`{"oldPassword": "OldPassword123!"}`), &dto)
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty object",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(`{}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				var dto domains.ChangePasswordDTO
				_ = json.Unmarshal([]byte(`{}`), &dto)
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - validation failed",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "old", NewPassword: "123"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "old", NewPassword: "123"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - wrong old password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrWrongPassword)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 999, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 999
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "different password strengths - strong password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "OldPassword123!", NewPassword: "NewSecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "NewSecurePassword123!"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different password strengths - very strong password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "OldPassword123!", NewPassword: "V3ry$tr0ngN3wP@ssw0rd!2024"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "V3ry$tr0ngN3wP@ssw0rd!2024"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different password strengths - weak password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "OldPassword123!", NewPassword: "123"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "123"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "different password strengths - common password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "OldPassword123!", NewPassword: "password"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "password"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "different password strengths - same as old password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "OldPassword123!", NewPassword: "OldPassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "OldPassword123!"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "JSON with extra fields",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				jsonWithExtraFields := `{
					"oldPassword": "OldPassword123!",
					"newPassword": "NewPassword123!",
					"extraField": "should be ignored",
					"anotherExtra": 123,
					"confirmPassword": "NewPassword123!"
				}`
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(jsonWithExtraFields)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "OldPassword123!", NewPassword: "NewPassword123!"}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "null values in JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				jsonWithNull := `{"oldPassword": null, "newPassword": null}`
				return createChangePasswordRequest(t, 123, bytes.NewReader([]byte(jsonWithNull)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "", NewPassword: ""}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "empty strings for passwords",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{OldPassword: "", NewPassword: ""}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.ChangePasswordDTO{UserID: 123, OldPassword: "", NewPassword: ""}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "zero user ID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 0, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 0
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "very large user ID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 18446744073709551615, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 18446744073709551615
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "status code 204 No Content on success",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := createChangePasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createChangePasswordDTO()
				dto.UserID = 123
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name: "user changes password for another user (should not be possible)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.ChangePasswordDTO{
					UserID:      456, // Другой пользователь
					OldPassword: "OldPassword123!",
					NewPassword: "NewPassword123!",
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createChangePasswordRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				// Ожидаем, что UserID в DTO будет перезаписан аутентифицированным userID
				dto := domains.ChangePasswordDTO{
					UserID:      123, // Должен быть перезаписан!
					OldPassword: "OldPassword123!",
					NewPassword: "NewPassword123!",
				}
				m.EXPECT().ChangePassword(gomock.Any(), dto).Return(customerrors.ErrWrongPassword)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)
			handler := change_password.Handler(mockUseCase)

			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	// Тесты, которые не вписываются в таблицу из-за сложной логики

	t.Run("concurrent password change requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := change_password.Handler(mockUseCase)

		const numRequests = 5

		userIDs := []uint64{1, 2, 3, 4, 5}
		oldPasswords := []string{"Pass1!", "Pass2!", "Pass3!", "Pass4!", "Pass5!"}
		newPasswords := []string{"New1!", "New2!", "New3!", "New4!", "New5!"}

		for i := range numRequests {
			expectedDTO := domains.ChangePasswordDTO{
				UserID:      userIDs[i],
				OldPassword: oldPasswords[i],
				NewPassword: newPasswords[i],
			}
			mockUseCase.EXPECT().ChangePassword(gomock.Any(), expectedDTO).Return(nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				dto := domains.ChangePasswordDTO{
					OldPassword: oldPasswords[idx],
					NewPassword: newPasswords[idx],
				}
				requestBody, _ := json.Marshal(dto)
				req := createChangePasswordRequest(t, userIDs[idx], bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				done <- true
			}(i)
		}

		for range numRequests {
			<-done
		}
	})

	t.Run("multiple password change attempts", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := change_password.Handler(mockUseCase)

		userID := uint64(123)
		dto := createChangePasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedDTO := dto
		expectedDTO.UserID = userID

		mockUseCase.EXPECT().
			ChangePassword(gomock.Any(), expectedDTO).
			Times(2).
			Return(nil)

		// Первый запрос
		req1 := createChangePasswordRequest(t, userID, bytes.NewReader(requestBody))
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй запрос (повторная смена пароля)
		req2 := createChangePasswordRequest(t, userID, bytes.NewReader(requestBody))
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		assert.Equal(t, http.StatusNoContent, rr2.Code)
	})
}
