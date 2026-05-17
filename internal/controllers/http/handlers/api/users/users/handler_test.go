package users_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	createRequest := func(t *testing.T, usernameFilter string, limit, offset int) *http.Request {
		t.Helper()

		url := "/users"
		queryParams := make([]string, 0)

		if usernameFilter != "" {
			queryParams = append(queryParams, "username="+usernameFilter)
		}

		if limit > 0 {
			queryParams = append(queryParams, "limit="+strconv.Itoa(limit))
		}

		if offset > 0 {
			queryParams = append(queryParams, "offset="+strconv.Itoa(offset))
		}

		if len(queryParams) > 0 {
			url += "?" + queryParams[0]

			var urlSb46 strings.Builder
			for i := 1; i < len(queryParams); i++ {
				urlSb46.WriteString("&" + queryParams[i])
			}

			url += urlSb46.String()
		}

		return httptest.NewRequest(http.MethodGet, url, http.NoBody)
	}

	createTestUsers := func(count int) []domains.User {
		now := time.Now()
		users := make([]domains.User, count)

		for i := range count {
			userID := uint64(i + 1)
			users[i] = domains.User{
				ID:             userID,
				Username:       "testuser" + strconv.FormatUint(userID, 10),
				Email:          "user" + strconv.FormatUint(userID, 10) + "@example.com",
				EmailConfirmed: i%2 == 0,
				Password:       "hashed_password_here",
				CreatedAt:      now.Add(time.Duration(i) * time.Hour),
				UpdatedAt:      now.Add(time.Duration(i) * time.Hour),
			}
		}

		return users
	}

	createTestUsersWithUsername := func(username string, count int) []domains.User {
		now := time.Now()
		users := make([]domains.User, count)

		for i := range count {
			userID := uint64(i + 1000)
			users[i] = domains.User{
				ID:             userID,
				Username:       username + strconv.Itoa(i+1),
				Email:          username + strconv.Itoa(i+1) + "@example.com",
				EmailConfirmed: true,
				Password:       "hashed_password_here",
				CreatedAt:      now.Add(time.Duration(i) * time.Hour),
				UpdatedAt:      now.Add(time.Duration(i) * time.Hour),
			}
		}

		return users
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get users without filters",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(3), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 3)

				firstUser := response[0]
				assert.Equal(t, float64(1), firstUser["id"])
				assert.Equal(t, "testuser1", firstUser["username"])
				assert.Equal(t, "user1@example.com", firstUser["email"])
				assert.True(t, firstUser["emailConfirmed"].(bool))
				assert.NotNil(t, firstUser["createdAt"])
				assert.NotNil(t, firstUser["updatedAt"])

				_, passwordExists := firstUser["password"]
				assert.False(t, passwordExists, "password should not be included in response")
			},
		},
		{
			name: "successful get users with username filter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "john", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{Username: pointers.New("john")}, nil).
					Return(createTestUsersWithUsername("john", 2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)

				for _, user := range response {
					username := user["username"].(string)
					assert.Contains(t, username, "john")
				}
			},
		},
		{
			name: "successful get users with pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 10, 20)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, &domains.Pagination{
						Limit:  pointers.New[uint64](10),
						Offset: pointers.New[uint64](20),
					}).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "successful get users with username filter and pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "alice", 5, 10)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(
						gomock.Any(),
						&domains.UsersFilters{Username: pointers.New("alice")},
						&domains.Pagination{
							Limit:  pointers.New[uint64](5),
							Offset: pointers.New[uint64](10),
						},
					).
					Return(createTestUsersWithUsername("alice", 3), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 3)
			},
		},
		{
			name: "successful get users - empty result",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return([]domains.User{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Empty(t, response)
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "pagination with invalid limit parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(
					http.MethodGet,
					"/users?limit=invalid&offset=0",
					http.NoBody,
				)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "pagination with invalid offset parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(
					http.MethodGet,
					"/users?limit=10&offset=invalid",
					http.NoBody,
				)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(
						gomock.Any(),
						nil,
						&domains.Pagination{
							Limit:  pointers.New[uint64](10),
							Offset: pointers.New[uint64](0),
						},
					).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "username filter with empty string",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(3), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 3)
			},
		},
		{
			name: "large number of users",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(100), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 100)
			},
		},
		{
			name: "GET method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodGet, "/users", http.NoBody)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "POST method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodPost, "/users", http.NoBody)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "PUT method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodPut, "/users", http.NoBody)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "DELETE method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodDelete, "/users", http.NoBody)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "PATCH method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodPatch, "/users", http.NoBody)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "multiple query parameters with same name",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(
					http.MethodGet,
					"/users?username=first&username=second",
					http.NoBody,
				)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), &domains.UsersFilters{Username: pointers.New("first")}, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "URL encoded username filter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(
					http.MethodGet,
					"/users?username=john%20doe%40example",
					http.NoBody,
				)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(
						gomock.Any(),
						&domains.UsersFilters{Username: pointers.New("john doe@example")},
						nil,
					).
					Return(createTestUsersWithUsername("john doe@example", 1), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 1)
				assert.Contains(t, response[0]["username"].(string), "john doe@example")
			},
		},
		{
			name: "max limit value",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 100, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, &domains.Pagination{
						Limit:  pointers.New[uint64](100),
						Offset: pointers.New[uint64](0),
					}).
					Return(createTestUsers(50), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 50)
			},
		},
		{
			name: "zero limit and offset",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(3), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 3)
			},
		},
		{
			name: "JSON encoding preserves structure",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "", 0, 0)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(createTestUsers(2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				responseBody := rr.Body.String()
				assert.True(t, responseBody != "")
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

			handler := users.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	t.Run("concurrent requests with different filters", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.Handler(mockUseCase)

		const numRequests = 5

		usernames := []string{"user1", "user2", "user3", "user4", "user5"}

		for _, username := range usernames {
			expectedUsers := createTestUsersWithUsername(username, 1)
			expectedFilters := &domains.UsersFilters{
				Username: pointers.New(username),
			}

			mockUseCase.EXPECT().
				GetUsers(gomock.Any(), expectedFilters, nil).
				Return(expectedUsers, nil)
		}

		done := make(chan bool, numRequests)

		for _, username := range usernames {
			go func(name string) {
				req := createRequest(t, name, 0, 0)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 1)
				assert.Contains(t, response[0]["username"].(string), name)

				done <- true
			}(username)
		}

		for range numRequests {
			<-done
		}
	})
}
