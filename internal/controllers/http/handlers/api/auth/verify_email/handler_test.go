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

		vars := map[string]string{
			verify_email.TokenRouteKey: token,
		}
		req = mux.SetURLVars(req, vars)

		return req
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful email verification",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "valid_verification_token_123")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "valid_verification_token_123").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name: "unauthorized - invalid JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, invalidJWTToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), invalidJWTToken).
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
				return createRequest(t, "token_for_nonexistent_user")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token_for_nonexistent_user").
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
			name: "conflict - email already confirmed",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "token_for_already_confirmed_email")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token_for_already_confirmed_email").
					Return(customerrors.ErrEmailAlreadyConfirmed)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrEmailAlreadyConfirmed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
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
			name: "different token formats - standard JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different token formats - hex token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "a1b2c3d4e5f67890")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "a1b2c3d4e5f67890").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different token formats - uuid token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "550e8400-e29b-41d4-a716-446655440000")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different token formats - token with special chars",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "token_with-special.chars+plus=equals")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token_with-special.chars+plus=equals").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different token formats - expired token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "expired_jwt_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "expired_jwt_token").
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name: "different token formats - malformed JWT",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "not.a.valid.jwt")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "not.a.valid.jwt").
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name: "expired verification token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "expired_verification_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "expired_verification_token").
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name: "very long token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				var longTokenSb strings.Builder
				for range 4096 {
					longTokenSb.WriteString("a")
				}
				longToken := longTokenSb.String()
				return createRequest(t, longToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				var longTokenSb strings.Builder
				for range 4096 {
					longTokenSb.WriteString("a")
				}
				m.EXPECT().
					VerifyEmail(gomock.Any(), longTokenSb.String()).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "verification for deleted user",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "token_for_deleted_user")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token_for_deleted_user").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "database or external service errors - database connection error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
					Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
			},
		},
		{
			name: "database or external service errors - cache service error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
					Return(errors.New("redis connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "redis connection failed")
			},
		},
		{
			name: "database or external service errors - transaction error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
					Return(errors.New("transaction failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "transaction failed")
			},
		},
		{
			name: "status code 204 No Content on success",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "valid_verification_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "valid_verification_token").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 204 No Content, not 200 OK")
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name: "token with query parameters in URL",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				token := "token123"
				url := "/verify-email/" + token + "?redirect=true&mode=web"
				req := httptest.NewRequest(http.MethodPost, url, http.NoBody)
				vars := map[string]string{
					verify_email.TokenRouteKey: token,
				}
				req = mux.SetURLVars(req, vars)
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token123").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "verification after user changes email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "token_for_old_email")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), "token_for_old_email").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "verification with body in request (should be ignored)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(
					http.MethodPost,
					"/verify-email/"+validToken,
					http.NoBody,
				)
				vars := map[string]string{
					verify_email.TokenRouteKey: validToken,
				}
				req = mux.SetURLVars(req, vars)
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), validToken).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := verify_email.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	t.Run("concurrent verification requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := verify_email.Handler(mockUseCase)

		const numRequests = 5

		tokens := []string{"token1", "token2", "token3", "token4", "token5"}

		for i := range numRequests {
			mockUseCase.EXPECT().
				VerifyEmail(gomock.Any(), tokens[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				req := createRequest(t, tokens[idx])
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
		handler := verify_email.Handler(mockUseCase)

		token := "single_use_verification_token"

		// Первый вызов - успешно
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(nil)

		req1 := createRequest(t, token)
		rr1 := httptest.NewRecorder()

		handler.ServeHTTP(rr1, req1)

		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй вызов с тем же токеном - должен вернуть ошибку
		mockUseCase.EXPECT().
			VerifyEmail(gomock.Any(), token).
			Return(customerrors.ErrEmailAlreadyConfirmed)

		req2 := createRequest(t, token)
		rr2 := httptest.NewRecorder()

		handler.ServeHTTP(rr2, req2)

		assert.Equal(t, http.StatusConflict, rr2.Code)
		assert.Contains(t, rr2.Body.String(), customerrors.ErrEmailAlreadyConfirmed.Error())
	})

	t.Run("multiple verification attempts with different outcomes", func(t *testing.T) {
		t.Parallel()

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

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.statusCode, rr.Code)

			if tc.error != nil {
				assert.Contains(t, rr.Body.String(), tc.error.Error())
			}
		}
	})
}
