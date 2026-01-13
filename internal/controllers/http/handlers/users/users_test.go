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

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/users"
	"github.com/DKhorkov/kfc/internal/domains"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUsersHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
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

	// Вспомогательная функция для создания тестовых пользователей
	createTestUsers := func(count int) []domains.User {
		now := time.Now()
		users := make([]domains.User, count)

		for i := range count {
			userID := uint64(i + 1)
			users[i] = domains.User{
				ID:             userID,
				Username:       "testuser" + strconv.FormatUint(userID, 10),
				Email:          "user" + strconv.FormatUint(userID, 10) + "@example.com",
				EmailConfirmed: i%2 == 0, // Каждый второй email не подтвержден
				Password:       "hashed_password_here",
				CreatedAt:      now.Add(time.Duration(i) * time.Hour),
				UpdatedAt:      now.Add(time.Duration(i) * time.Hour),
			}
		}

		return users
	}

	// Вспомогательная функция для создания пользователей с определенным username
	createTestUsersWithUsername := func(username string, count int) []domains.User {
		now := time.Now()
		users := make([]domains.User, count)

		for i := range count {
			userID := uint64(i + 1000) // Начинаем с 1000 чтобы не пересекаться
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

	t.Run("successful get users without filters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(3)

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 3)

		// Проверяем структуру первого пользователя
		firstUser := response[0]
		assert.Equal(t, float64(1), firstUser["id"])
		assert.Equal(t, "testuser1", firstUser["username"])
		assert.Equal(t, "user1@example.com", firstUser["email"])
		assert.True(t, firstUser["emailConfirmed"].(bool))
		assert.NotNil(t, firstUser["createdAt"])
		assert.NotNil(t, firstUser["updatedAt"])

		// Проверяем, что пароль не возвращается
		_, passwordExists := firstUser["password"]
		assert.False(t, passwordExists, "password should not be included in response")
	})

	t.Run("successful get users with username filter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		usernameFilter := "john"
		expectedUsers := createTestUsersWithUsername(usernameFilter, 2)

		expectedFilters := &domains.UsersFilters{
			Username: pointers.New(usernameFilter),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), expectedFilters, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, usernameFilter, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)

		// Проверяем, что все пользователи содержат искомый username
		for _, user := range response {
			username := user["username"].(string)
			assert.Contains(t, username, usernameFilter)
		}
	})

	t.Run("successful get users with pagination", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(2)

		expectedPagination := &domains.Pagination{
			Limit:  pointers.New[uint64](10),
			Offset: pointers.New[uint64](20),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, expectedPagination).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 10, 20)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("successful get users with username filter and pagination", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		usernameFilter := "alice"
		expectedUsers := createTestUsersWithUsername(usernameFilter, 3)

		expectedFilters := &domains.UsersFilters{
			Username: pointers.New(usernameFilter),
		}

		expectedPagination := &domains.Pagination{
			Limit:  pointers.New[uint64](5),
			Offset: pointers.New[uint64](10),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), expectedFilters, expectedPagination).
			Return(expectedUsers, nil)

		req := createRequest(t, usernameFilter, 5, 10)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 3)
	})

	t.Run("successful get users - empty result", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return([]domains.User{}, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("pagination with invalid limit parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(2)

		// При невалидном limit, GetPaginationFromRequest должен вернуть nil
		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		// Запрос с невалидным limit
		req := httptest.NewRequest(http.MethodGet, "/users?limit=invalid&offset=0", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен обработаться с дефолтными значениями
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("pagination with invalid offset parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(2)

		mockUseCase.EXPECT().
			GetUsers(
				gomock.Any(),
				nil,
				&domains.Pagination{
					Limit:  pointers.New[uint64](10),
					Offset: pointers.New[uint64](0),
				},
			).
			Return(expectedUsers, nil)

		// Запрос с невалидным offset
		req := httptest.NewRequest(http.MethodGet, "/users?limit=10&offset=invalid", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен обработаться с дефолтными значениями
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("username filter with empty string", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(3)

		// При пустом username фильтр не должен создаваться
		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 3)
	})

	t.Run("large number of users", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		const userCount = 100

		expectedUsers := createTestUsers(userCount)

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, userCount)
	})

	t.Run("different HTTP methods", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name           string
			method         string
			expectedStatus int
		}{
			{"GET method", http.MethodGet, http.StatusOK},
			{"POST method", http.MethodPost, http.StatusOK}, // Обработчик не проверяет метод
			{"PUT method", http.MethodPut, http.StatusOK},
			{"DELETE method", http.MethodDelete, http.StatusOK},
			{"PATCH method", http.MethodPatch, http.StatusOK},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
				handler := users.GetUsersHandler(mockUseCase)

				expectedUsers := createTestUsers(2)

				mockUseCase.EXPECT().
					GetUsers(gomock.Any(), nil, nil).
					Return(expectedUsers, nil)

				req := httptest.NewRequest(tc.method, "/users", http.NoBody)
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, tc.expectedStatus, rr.Code)
			})
		}
	})

	t.Run("multiple query parameters with same name", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(2)

		// Должен взять первое значение username
		expectedFilters := &domains.UsersFilters{
			Username: pointers.New("first"),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), expectedFilters, nil).
			Return(expectedUsers, nil)

		// Запрос с дублирующимися параметрами username
		req := httptest.NewRequest(
			http.MethodGet,
			"/users?username=first&username=second",
			http.NoBody,
		)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("concurrent requests with different filters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 5

		usernames := []string{"user1", "user2", "user3", "user4", "user5"}

		// Настраиваем мок на вызовы с разными фильтрами
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

		// Act - запускаем concurrent запросы
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

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("URL encoded username filter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		// Username с пробелами и специальными символами
		encodedUsername := "john%20doe%40example"
		decodedUsername := "john doe@example"

		expectedUsers := createTestUsersWithUsername(decodedUsername, 1)
		expectedFilters := &domains.UsersFilters{
			Username: pointers.New(decodedUsername),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), expectedFilters, nil).
			Return(expectedUsers, nil)

		// Запрос с URL-encoded username
		req := httptest.NewRequest(http.MethodGet, "/users?username="+encodedUsername, http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 1)

		// Username в ответе должен соответствовать декодированному значению
		assert.Contains(t, response[0]["username"].(string), decodedUsername)
	})

	t.Run("max limit value", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(50)

		expectedPagination := &domains.Pagination{
			Limit:  pointers.New[uint64](100), // Максимальный лимит
			Offset: pointers.New[uint64](0),
		}

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, expectedPagination).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 100, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 50)
	})

	t.Run("zero limit and offset", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(3)

		// При limit=0, GetPaginationFromRequest должен вернуть nil
		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 3)
	})

	t.Run("JSON encoding preserves structure", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := users.GetUsersHandler(mockUseCase)

		expectedUsers := createTestUsers(2)

		mockUseCase.EXPECT().
			GetUsers(gomock.Any(), nil, nil).
			Return(expectedUsers, nil)

		req := createRequest(t, "", 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем, что JSON содержит правильные поля
		responseBody := rr.Body.String()

		// Должен быть массив
		assert.True(t, responseBody != "")

		// Проверяем наличие обязательных полей
		assert.Contains(t, responseBody, `"id"`)
		assert.Contains(t, responseBody, `"username"`)
		assert.Contains(t, responseBody, `"email"`)
		assert.Contains(t, responseBody, `"emailConfirmed"`)
		assert.Contains(t, responseBody, `"createdAt"`)
		assert.Contains(t, responseBody, `"updatedAt"`)

		// Проверяем, что пароль не возвращается
		assert.NotContains(t, responseBody, `"password"`)
	})
}
