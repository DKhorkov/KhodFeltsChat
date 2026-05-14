package send_forget_password_message_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/send_forget_password_message"
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

		return httptest.NewRequest(http.MethodPost, "/forget-password/send", requestBody)
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful send forget password message",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
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
				return createRequest(t, bytes.NewReader([]byte{}))
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
				return createRequest(t, bytes.NewReader([]byte(`{"email": invalid}`)))
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
				missingFieldJSON := `{}`
				var dto domains.SendForgetPasswordMessageDTO
				err := json.Unmarshal([]byte(missingFieldJSON), &dto)
				require.NoError(t, err)
				assert.Equal(t, "", dto.Email)
				return createRequest(t, bytes.NewReader([]byte(missingFieldJSON)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "").
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
			name: "bad request - empty email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: ""}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "").
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
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
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
			name: "forbidden - email not confirmed",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(customerrors.ErrEmailNotConfirmed)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrEmailNotConfirmed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("email service unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "email service unavailable")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "different email formats - simple email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different email formats - email with plus",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user+tag@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user+tag@example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different email formats - email with dot",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user.name@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user.name@example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different email formats - email with subdomain",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@sub.example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@sub.example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "different email formats - invalid email format",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "not-an-email"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "not-an-email").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "different email formats - email with spaces",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user @example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user @example.com").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "different email formats - case insensitive email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "USER@EXAMPLE.COM"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "USER@EXAMPLE.COM").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "JSON with extra fields",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				jsonWithExtraFields := `{
				"email": "user@example.com",
				"extraField": "should be ignored",
				"anotherExtra": 123,
				"username": "user123"
			}`
				return createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "null email in JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				jsonWithNull := `{"email": null}`
				var dto domains.SendForgetPasswordMessageDTO
				err := json.Unmarshal([]byte(jsonWithNull), &dto)
				require.NoError(t, err)
				assert.Equal(t, "", dto.Email)
				return createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "email with special characters",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user.name+tag@example-domain.co.uk"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user.name+tag@example-domain.co.uk").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "non-existent domain email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@nonexistent-domain-12345.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@nonexistent-domain-12345.com").
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name: "status code 204 No Content on success",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
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
			name: "case insensitive email matching - uppercase",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "USER@EXAMPLE.COM"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "USER@EXAMPLE.COM").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "case insensitive email matching - mixed case",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "User@Example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "User@Example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "case insensitive email matching - lowercase",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "email with international characters",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@müller.de"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@müller.de").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "database or external service errors - database connection error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
			},
		},
		{
			name: "database or external service errors - email service error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("SMTP server unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "SMTP server unavailable")
			},
		},
		{
			name: "database or external service errors - rate limit exceeded in service",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				dto := domains.SendForgetPasswordMessageDTO{Email: "user@example.com"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)
				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					SendForgetPasswordMessage(gomock.Any(), "user@example.com").
					Return(errors.New("rate limit exceeded"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "rate limit exceeded")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := send_forget_password_message.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	t.Run("concurrent requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := send_forget_password_message.Handler(mockUseCase)

		const numRequests = 5

		emails := []string{
			"user1@example.com",
			"user2@example.com",
			"user3@example.com",
			"user4@example.com",
			"user5@example.com",
		}

		for i := range numRequests {
			mockUseCase.EXPECT().
				SendForgetPasswordMessage(gomock.Any(), emails[i]).
				Return(nil)
		}

		done := make(chan bool, numRequests)

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

		for range numRequests {
			<-done
		}
	})

	t.Run("rate limiting scenarios", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := send_forget_password_message.Handler(mockUseCase)

		dto := domains.SendForgetPasswordMessageDTO{
			Email: "user@example.com",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		const attempts = 3
		for range attempts {
			mockUseCase.EXPECT().
				SendForgetPasswordMessage(gomock.Any(), dto.Email).
				Return(nil)
		}

		for range attempts {
			req := createRequest(t, bytes.NewReader(requestBody))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusNoContent, rr.Code)
		}
	})
}
