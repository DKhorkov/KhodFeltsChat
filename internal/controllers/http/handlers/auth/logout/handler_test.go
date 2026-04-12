package logout_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/logout"
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

		req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	t.Run("successful logout", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())

		// Проверяем, что установлены пустые куки с MaxAge = -1
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)

		// Находим access token cookie
		var (
			accessTokenCookie  *http.Cookie
			refreshTokenCookie *http.Cookie
		)

		for _, cookie := range cookies {
			switch cookie.Name {
			case login.AccessTokenCookieName:
				accessTokenCookie = cookie
			case login.RefreshTokenCookieName:
				refreshTokenCookie = cookie
			}
		}

		require.NotNil(t, accessTokenCookie)
		require.NotNil(t, refreshTokenCookie)

		// Проверяем, что куки очищены
		assert.Equal(t, "", accessTokenCookie.Value)
		assert.Equal(t, "", refreshTokenCookie.Value)
		assert.Equal(t, -1, accessTokenCookie.MaxAge)
		assert.Equal(t, -1, refreshTokenCookie.MaxAge)

		// Проверяем другие настройки кук (должны быть дефолтными)
		assert.Equal(t, "", accessTokenCookie.Path)
		assert.Equal(t, "", refreshTokenCookie.Path)
		assert.False(t, accessTokenCookie.Secure) // Config по умолчанию
		assert.False(t, refreshTokenCookie.Secure)
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		// Запрос БЕЗ userID в контексте
		req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		cookies := rr.Result().Cookies()
		assert.Empty(t, cookies, "Should not set cookies on unauthorized")
	})

	t.Run("unauthorized - invalid userID type in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		// Запрос с userID неверного типа в контексте
		req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, "not-a-number")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		cookies := rr.Result().Cookies()
		assert.Empty(t, cookies)
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(errors.New("database connection failed"))

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены при ошибке
		cookies := rr.Result().Cookies()
		assert.Empty(t, cookies, "Should not set cookies on error")
	})

	t.Run("zero user ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(0) // Нулевой ID

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Проверяем, что куки очищены
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)

		for _, cookie := range cookies {
			assert.Equal(t, "", cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
		}
	})

	t.Run("logout with existing cookies - should clear them", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)

		// Добавляем существующие куки (должны быть очищены)
		req.AddCookie(&http.Cookie{
			Name:  login.AccessTokenCookieName,
			Value: "existing_access_token",
		})
		req.AddCookie(&http.Cookie{
			Name:  login.RefreshTokenCookieName,
			Value: "existing_refresh_token",
		})

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Проверяем, что куки очищены
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)

		for _, cookie := range cookies {
			assert.Equal(t, "", cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
		}
	})

	t.Run("different HTTP methods", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name           string
			method         string
			expectedStatus int
		}{
			{"POST method", http.MethodPost, http.StatusNoContent},
			{"GET method", http.MethodGet, http.StatusNoContent}, // Обработчик не проверяет метод
			{"PUT method", http.MethodPut, http.StatusNoContent},
			{"DELETE method", http.MethodDelete, http.StatusNoContent},
			{"PATCH method", http.MethodPatch, http.StatusNoContent},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := logout.Handler(mockUseCase)

				userID := uint64(123)

				mockUseCase.EXPECT().
					LogoutUser(gomock.Any(), userID).
					Return(nil)

				req := httptest.NewRequest(tc.method, "/logout", http.NoBody)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
				req = req.WithContext(ctx)
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, tc.expectedStatus, rr.Code)

				if tc.expectedStatus == http.StatusNoContent {
					// Проверяем, что куки очищены
					cookies := rr.Result().Cookies()
					assert.Len(t, cookies, 2)
				}
			})
		}
	})

	t.Run("concurrent logout requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		const numRequests = 10

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			userID := uint64(i + 1)
			mockUseCase.EXPECT().
				LogoutUser(gomock.Any(), userID).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				userID := uint64(idx + 1)
				req := createRequest(t, userID)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				// Проверяем, что куки очищены
				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

				for _, cookie := range cookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("logout after successful use case but cookie set fails", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)

		// Используем ResponseWriter, который записывает в буфер
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - даже если установка кук "не удалась", статус должен быть 204
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Проверяем, что все равно пытались очистить куки
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)

		for _, cookie := range cookies {
			assert.Equal(t, "", cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
		}
	})

	t.Run("logout invalidates user session in use case", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		// Проверяем, что LogoutUser вызывается с правильным userID
		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			DoAndReturn(func(_ any, id uint64) error {
				assert.Equal(t, userID, id)

				return nil
			})

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("logout with very large user ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(18446744073709551615) // Максимальный uint64

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("logout when user doesn't exist", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(999) // Несуществующий пользователь

		// LogoutUser может успешно выполниться даже для несуществующего пользователя
		// (например, просто ничего не делать)
		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Куки все равно должны быть очищены
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)
	})

	t.Run("multiple logout calls for same user", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		// Ожидаем два вызова LogoutUser для одного пользователя
		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Times(2).
			Return(nil)

		// Первый запрос
		req1 := createRequest(t, userID)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)

		// Assert - первый запрос
		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй запрос (повторный logout)
		req2 := createRequest(t, userID)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

		// Assert - второй запрос
		assert.Equal(t, http.StatusNoContent, rr2.Code)

		// Оба раза куки должны быть очищены
		for _, rr := range []*httptest.ResponseRecorder{rr1, rr2} {
			cookies := rr.Result().Cookies()
			assert.Len(t, cookies, 2)

			for _, cookie := range cookies {
				assert.Equal(t, "", cookie.Value)
				assert.Equal(t, -1, cookie.MaxAge)
			}
		}
	})

	t.Run("logout with additional cookies present", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Return(nil)

		req := createRequest(t, userID)

		// Добавляем дополнительные куки (не должны влиять на обработку)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
		req.AddCookie(&http.Cookie{Name: "preferences", Value: "theme=dark"})

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Проверяем, что очищены только токен-куки
		cookies := rr.Result().Cookies()

		// Находим токен-куки
		var (
			tokenCookies []*http.Cookie
			otherCookies []*http.Cookie
		)

		for _, cookie := range cookies {
			if cookie.Name == login.AccessTokenCookieName ||
				cookie.Name == login.RefreshTokenCookieName {
				tokenCookies = append(tokenCookies, cookie)
			} else {
				otherCookies = append(otherCookies, cookie)
			}
		}

		// Токен-куки должны быть очищены
		assert.Len(t, tokenCookies, 2)

		for _, cookie := range tokenCookies {
			assert.Equal(t, "", cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
		}

		// Другие куки не должны быть затронуты (но их и не должно быть в ответе)
		assert.Empty(t, otherCookies)
	})
}
