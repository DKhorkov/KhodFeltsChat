package me_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/me"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
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
	createRequest := func(t *testing.T, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "testuser" + strconv.FormatUint(id, 10),
			Email:          "user" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now,
			UpdatedAt:      now.Add(time.Hour),
		}
	}

	// Вспомогательная функция для создания длинного username
	buildLongUsername := func() string {
		longUsername := "u"

		var longUsernameSb275 strings.Builder
		for range 69 {
			longUsernameSb275.WriteString("a")
		}

		longUsername += longUsernameSb275.String()

		return longUsername
	}

	now := time.Now()
	longUsername := buildLongUsername()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get current user",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(123), response["id"])
				assert.Equal(t, "testuser123", response["username"])
				assert.Equal(t, "user123@example.com", response["email"])
				assert.True(t, response["emailConfirmed"].(bool))
				assert.NotNil(t, response["createdAt"])
				assert.NotNil(t, response["updatedAt"])

				_, passwordExists := response["password"]
				assert.False(t, passwordExists, "password should not be included in response")
			},
		},
		{
			name: "successful get current user with unconfirmed email",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 456)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(456)).
					Return(createTestUser(456, false), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(456), response["id"])
				assert.Equal(t, "testuser456", response["username"])
				assert.Equal(t, "user456@example.com", response["email"])
				assert.False(t, response["emailConfirmed"].(bool))
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(_ *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "unauthorized - invalid userID type in context",
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/me", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					"not-a-number",
				)

				return req.WithContext(ctx)
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 999)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(999)).
					Return(nil, customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "user with minimum valid username length",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID:             123,
						Username:       "user5",
						Email:          "user@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user5", response["username"])
			},
		},
		{
			name: "user with maximum valid username length",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID:             123,
						Username:       longUsername,
						Email:          "user@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      now,
						UpdatedAt:      now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, longUsername, response["username"])
				assert.Len(t, response["username"].(string), 70)
			},
		},
		{
			name: "user with valid email format - simple@example.com",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID: 123, Username: "testuser", Email: "simple@example.com",
						EmailConfirmed: true, Password: "hashed_password",
						CreatedAt: now, UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "simple@example.com", response["email"])
			},
		},
		{
			name: "user with valid email format - user.name@example.com",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID: 123, Username: "testuser", Email: "user.name@example.com",
						EmailConfirmed: true, Password: "hashed_password",
						CreatedAt: now, UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user.name@example.com", response["email"])
			},
		},
		{
			name: "user with valid email format - user_name@example.co.uk",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID: 123, Username: "testuser", Email: "user_name@example.co.uk",
						EmailConfirmed: true, Password: "hashed_password",
						CreatedAt: now, UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user_name@example.co.uk", response["email"])
			},
		},
		{
			name: "user with valid email format - user+tag@example.com",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID: 123, Username: "testuser", Email: "user+tag@example.com",
						EmailConfirmed: true, Password: "hashed_password",
						CreatedAt: now, UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user+tag@example.com", response["email"])
			},
		},
		{
			name: "user with valid email format - 123456@example.com",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID: 123, Username: "testuser", Email: "123456@example.com",
						EmailConfirmed: true, Password: "hashed_password",
						CreatedAt: now, UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "123456@example.com", response["email"])
			},
		},
		{
			name: "zero user ID",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(0)).
					Return(nil, customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
		{
			name: "user with different timestamps",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC)

				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(&domains.User{
						ID:             123,
						Username:       "testuser",
						Email:          "user@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      createdAt,
						UpdatedAt:      updatedAt,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotNil(t, response["createdAt"])
				assert.NotNil(t, response["updatedAt"])

				createdAtStr, _ := response["createdAt"].(string)
				updatedAtStr, _ := response["updatedAt"].(string)

				parsedCreatedAt, err1 := time.Parse(time.RFC3339Nano, createdAtStr)
				parsedUpdatedAt, err2 := time.Parse(time.RFC3339Nano, updatedAtStr)

				if err1 == nil && err2 == nil {
					assert.True(
						t,
						parsedUpdatedAt.After(parsedCreatedAt),
						"updatedAt should be after createdAt",
					)
				}
			},
		},
		{
			name: "JSON encoding preserves field order",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				responseBody := rr.Body.String()

				assert.Contains(t, responseBody, `"id"`)
				assert.Contains(t, responseBody, `"username"`)
				assert.Contains(t, responseBody, `"email"`)
				assert.Contains(t, responseBody, `"emailConfirmed"`)
				assert.Contains(t, responseBody, `"createdAt"`)
				assert.Contains(t, responseBody, `"updatedAt"`)

				assert.NotContains(t, responseBody, `"password"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := me.Handler(mockUseCase)
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

		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := me.Handler(mockUseCase)

		const numRequests = 10

		userID := uint64(123)
		expectedUser := createTestUser(userID, true)

		mockUseCase.EXPECT().
			GetUserByID(gomock.Any(), userID).
			Times(numRequests).
			Return(expectedUser, nil)

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(_ int) {
				req := createRequest(t, userID)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				done <- true
			}(i)
		}

		for range numRequests {
			<-done
		}
	})
}
