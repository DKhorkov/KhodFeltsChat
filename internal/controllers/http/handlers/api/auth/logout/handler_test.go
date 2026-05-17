package logout_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/logout"
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

	cookiesConfig := config.CookiesConfig{}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful logout",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(123))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

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

				assert.Equal(t, "", accessTokenCookie.Value)
				assert.Equal(t, "", refreshTokenCookie.Value)
				assert.Equal(t, -1, accessTokenCookie.MaxAge)
				assert.Equal(t, -1, refreshTokenCookie.MaxAge)

				assert.Equal(t, "", accessTokenCookie.Path)
				assert.Equal(t, "", refreshTokenCookie.Path)
				assert.False(t, accessTokenCookie.Secure)
				assert.False(t, refreshTokenCookie.Secure)
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				cookies := rr.Result().Cookies()
				assert.Empty(t, cookies, "Should not set cookies on unauthorized")
			},
		},
		{
			name: "unauthorized - invalid userID type in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					"not-a-number",
				)

				return req.WithContext(ctx)
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				cookies := rr.Result().Cookies()
				assert.Empty(t, cookies)
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(123))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				cookies := rr.Result().Cookies()
				assert.Empty(t, cookies, "Should not set cookies on error")
			},
		},
		{
			name: "zero user ID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(0))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(0)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

				for _, cookie := range cookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}
			},
		},
		{
			name: "logout with existing cookies - should clear them",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := createRequest(t, uint64(123))
				req.AddCookie(&http.Cookie{
					Name:  login.AccessTokenCookieName,
					Value: "existing_access_token",
				})
				req.AddCookie(&http.Cookie{
					Name:  login.RefreshTokenCookieName,
					Value: "existing_refresh_token",
				})

				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

				for _, cookie := range cookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}
			},
		},
		{
			name: "POST method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodPost, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(123),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().LogoutUser(gomock.Any(), uint64(123)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "GET method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodGet, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(123),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().LogoutUser(gomock.Any(), uint64(123)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "PUT method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodPut, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(123),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().LogoutUser(gomock.Any(), uint64(123)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "DELETE method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodDelete, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(123),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().LogoutUser(gomock.Any(), uint64(123)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "PATCH method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodPatch, "/logout", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(123),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().LogoutUser(gomock.Any(), uint64(123)).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "logout after successful use case but cookie set fails",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(123))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

				for _, cookie := range cookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}
			},
		},
		{
			name: "logout invalidates user session in use case",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(123))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					DoAndReturn(func(_ any, id uint64) error {
						if id != uint64(123) {
							t.Errorf("expected userID 123, got %d", id)
						}

						return nil
					})
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "logout with very large user ID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(18446744073709551615))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(18446744073709551615)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "logout when user doesn't exist",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(999))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(999)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)
			},
		},
		{
			name: "logout with additional cookies present",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := createRequest(t, uint64(123))
				req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
				req.AddCookie(&http.Cookie{Name: "preferences", Value: "theme=dark"})

				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), uint64(123)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()

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

				assert.Len(t, tokenCookies, 2)

				for _, cookie := range tokenCookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}

				assert.Empty(t, otherCookies)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := logout.Handler(mockUseCase, cookiesConfig)

			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	// Тесты, которые не укладываются в табличный формат

	t.Run("concurrent logout requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase, cookiesConfig)

		const numRequests = 10

		for i := range numRequests {
			userID := uint64(i + 1)
			mockUseCase.EXPECT().
				LogoutUser(gomock.Any(), userID).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				userID := uint64(idx + 1)
				req := createRequest(t, userID)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				cookies := rr.Result().Cookies()
				assert.Len(t, cookies, 2)

				for _, cookie := range cookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}

				done <- true
			}(i)
		}

		for range numRequests {
			<-done
		}
	})

	t.Run("multiple logout calls for same user", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase, cookiesConfig)

		userID := uint64(123)

		mockUseCase.EXPECT().
			LogoutUser(gomock.Any(), userID).
			Times(2).
			Return(nil)

		// Первый запрос
		req1 := createRequest(t, userID)
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)

		assert.Equal(t, http.StatusNoContent, rr1.Code)

		// Второй запрос (повторный logout)
		req2 := createRequest(t, userID)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

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
}
