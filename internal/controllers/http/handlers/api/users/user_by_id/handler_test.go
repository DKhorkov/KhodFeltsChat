package user_by_id_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/user_by_id"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
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

		req := httptest.NewRequest(
			http.MethodGet,
			"/users/"+strconv.FormatUint(userID, 10),
			http.NoBody,
		)

		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(userID, 10),
		}
		req = mux.SetURLVars(req, vars)

		return req
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

	now := time.Now()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get user by ID",
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
			name: "successful get user by ID with unconfirmed email",
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
			name: "bad request - invalid user ID parameter",
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/users/invalid", http.NoBody)
				vars := map[string]string{
					common.IDRouteKey: "invalid",
				}

				return mux.SetURLVars(req, vars)
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Contains(t, rr.Body.String(), "invalid syntax")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty user ID parameter",
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/users/", http.NoBody)
				vars := map[string]string{
					common.IDRouteKey: "",
				}

				return mux.SetURLVars(req, vars)
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Contains(t, rr.Body.String(), "invalid syntax")
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
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
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
			name: "user with minimum valid ID",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 1)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(1)).
					Return(createTestUser(1, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(1), response["id"])
			},
		},
		{
			name: "user with very large ID",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 18446744073709551615)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				userID := uint64(18446744073709551615)
				m.EXPECT().
					GetUserByID(gomock.Any(), userID).
					Return(createTestUser(userID, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, 1.8446744073709552e+19, response["id"])
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
			name: "request with query parameters",
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest(
					http.MethodGet,
					"/users/123?fields=all&expand=true",
					http.NoBody,
				)
				vars := map[string]string{
					common.IDRouteKey: "123",
				}

				return mux.SetURLVars(req, vars)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(123)).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
		{
			name: "request with headers",
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/users/123", http.NoBody)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Accept-Language", "en-US")
				req.Header.Set("User-Agent", "TestClient/1.0")

				vars := map[string]string{
					common.IDRouteKey: "123",
				}

				return mux.SetURLVars(req, vars)
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
			},
		},
		{
			name: "zero user ID - should fail parsing",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUserByID(gomock.Any(), uint64(0)).
					Return(createTestUser(0, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
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

			handler := user_by_id.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	t.Run("concurrent requests for different users", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := user_by_id.Handler(mockUseCase)

		const numRequests = 5

		userIDs := []uint64{1, 2, 3, 4, 5}

		for _, userID := range userIDs {
			expectedUser := createTestUser(userID, true)
			mockUseCase.EXPECT().
				GetUserByID(gomock.Any(), userID).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		for _, userID := range userIDs {
			go func(id uint64) {
				req := createRequest(t, id)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, float64(id), response["id"])

				done <- true
			}(userID)
		}

		for range numRequests {
			<-done
		}
	})
}
