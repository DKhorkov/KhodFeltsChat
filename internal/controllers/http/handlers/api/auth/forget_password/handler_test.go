package forget_password_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/forget_password"
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
	validToken       = "valid_token"
)

// Вспомогательная функция для создания запроса.
func createForgetPasswordRequest(t *testing.T, token string, requestBody io.Reader) *http.Request {
	t.Helper()

	url := "/forget-password/" + token
	req := httptest.NewRequest(http.MethodPost, url, requestBody)

	vars := map[string]string{
		forget_password.TokenRouteKey: token,
	}
	req = mux.SetURLVars(req, vars)

	return req
}

// Вспомогательная функция для создания ForgetPasswordDTO.
func createForgetPasswordDTO() domains.ForgetPasswordDTO {
	return domains.ForgetPasswordDTO{
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
			name: "successful password reset",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(
					t,
					"valid_reset_token_123",
					bytes.NewReader(requestBody),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), "valid_reset_token_123", dto.NewPassword).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name: "bad request - empty request body",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createForgetPasswordRequest(t, validToken, bytes.NewReader([]byte{}))
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

				invalidJSON := `{"newPassword": invalid}`

				return createForgetPasswordRequest(
					t,
					validToken,
					bytes.NewReader([]byte(invalidJSON)),
				)
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
			name: "bad request - missing required field",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createForgetPasswordRequest(
					t,
					validToken,
					bytes.NewReader([]byte(missingFieldJSON)),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "").
					Return(customerrors.ErrValidationFailed)
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

				dto := domains.ForgetPasswordDTO{NewPassword: "123"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "123").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "unauthorized - invalid JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(
					t,
					"invalid_jwt_token",
					bytes.NewReader(requestBody),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), "invalid_jwt_token", dto.NewPassword).
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(
					t,
					"valid_token_for_nonexistent_user",
					bytes.NewReader(requestBody),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), "valid_token_for_nonexistent_user", dto.NewPassword).
					Return(customerrors.ErrUserNotFound)
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

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, dto.NewPassword).
					Return(errors.New("database connection failed"))
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

				dto := domains.ForgetPasswordDTO{NewPassword: "NewSecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "NewSecurePassword123!").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different password strengths - very strong password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.ForgetPasswordDTO{NewPassword: "V3ry$tr0ngN3wP@ssw0rd!2024"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "V3ry$tr0ngN3wP@ssw0rd!2024").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different password strengths - weak password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.ForgetPasswordDTO{NewPassword: "123"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "123").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "different password strengths - common password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.ForgetPasswordDTO{NewPassword: "password"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "password").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "different password strengths - empty password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.ForgetPasswordDTO{NewPassword: ""}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "JSON with extra fields",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				jsonWithExtraFields := `{
					"newPassword": "NewSecurePassword123!",
					"extraField": "should be ignored",
					"anotherExtra": 123,
					"confirmPassword": "NewSecurePassword123!"
				}`

				return createForgetPasswordRequest(
					t,
					validToken,
					bytes.NewReader([]byte(jsonWithExtraFields)),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "NewSecurePassword123!").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "null password in JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createForgetPasswordRequest(
					t,
					validToken,
					bytes.NewReader([]byte(`{"newPassword": null}`)),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "expired token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, "expired_token", bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), "expired_token", dto.NewPassword).
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name: "malformed JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(
					t,
					"not.a.valid.jwt.token",
					bytes.NewReader(requestBody),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().
					ForgetPassword(gomock.Any(), "not.a.valid.jwt.token", dto.NewPassword).
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name: "same password as old password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.ForgetPasswordDTO{NewPassword: "SameAsOldPassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validToken, "SameAsOldPassword123!").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "status code 204 No Content on success",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createForgetPasswordDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createForgetPasswordRequest(t, validToken, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := createForgetPasswordDTO()
				m.EXPECT().ForgetPassword(gomock.Any(), validToken, dto.NewPassword).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.NotEqual(
					t,
					http.StatusOK,
					rr.Code,
					"Should return 204 No Content, not 200 OK",
				)
				assert.Empty(t, rr.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)
			handler := forget_password.Handler(mockUseCase)

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

	t.Run("concurrent password reset requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := forget_password.Handler(mockUseCase)

		const numRequests = 5

		tokens := []string{"token1", "token2", "token3", "token4", "token5"}
		passwords := []string{"Pass1!", "Pass2!", "Pass3!", "Pass4!", "Pass5!"}

		for i := range numRequests {
			mockUseCase.EXPECT().
				ForgetPassword(gomock.Any(), tokens[i], passwords[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				dto := domains.ForgetPasswordDTO{
					NewPassword: passwords[idx],
				}
				requestBody, _ := json.Marshal(dto)
				req := createForgetPasswordRequest(t, tokens[idx], bytes.NewReader(requestBody))
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

	t.Run("token used multiple times", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := forget_password.Handler(mockUseCase)

		token := "single_use_token"
		dto := createForgetPasswordDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Первый вызов - успешно
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(nil)

		req1 := createForgetPasswordRequest(t, token, bytes.NewReader(requestBody))
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй вызов с тем же токеном - должен вернуть ошибку
		mockUseCase.EXPECT().
			ForgetPassword(gomock.Any(), token, dto.NewPassword).
			Return(customerrors.ErrInvalidJWT)

		req2 := createForgetPasswordRequest(t, token, bytes.NewReader(requestBody))
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)
		assert.Equal(t, http.StatusUnauthorized, rr2.Code)
		assert.Contains(t, rr2.Body.String(), customerrors.ErrInvalidJWT.Error())
	})
}
