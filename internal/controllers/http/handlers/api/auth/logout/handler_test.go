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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	cookiesConfig := config.CookiesConfig{}

	// Вспомогательная функция для создания запроса с refresh token cookie
	createRequest := func(t *testing.T, refreshToken string) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodDelete, "/sessions", http.NoBody)
		req.AddCookie(&http.Cookie{
			Name:  login.RefreshTokenCookieName,
			Value: refreshToken,
		})

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
			name: "successful logout",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "encoded_refresh_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), "encoded_refresh_token").
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
			},
		},
		{
			name: "unauthorized - no refresh token cookie",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodDelete, "/sessions", http.NoBody)
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "named cookie not present")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				cookies := rr.Result().Cookies()
				assert.Empty(t, cookies, "Should not set cookies on unauthorized")
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "encoded_refresh_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), "encoded_refresh_token").
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
			name: "logout with empty refresh token cookie",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, "")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), "").
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
			name: "logout with additional cookies present",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := createRequest(t, "encoded_refresh_token")
				req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})
				req.AddCookie(&http.Cookie{Name: "preferences", Value: "theme=dark"})

				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LogoutUser(gomock.Any(), "encoded_refresh_token").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				cookies := rr.Result().Cookies()

				var tokenCookies []*http.Cookie

				for _, cookie := range cookies {
					if cookie.Name == login.AccessTokenCookieName ||
						cookie.Name == login.RefreshTokenCookieName {
						tokenCookies = append(tokenCookies, cookie)
					}
				}

				assert.Len(t, tokenCookies, 2)

				for _, cookie := range tokenCookies {
					assert.Equal(t, "", cookie.Value)
					assert.Equal(t, -1, cookie.MaxAge)
				}
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

	t.Run("concurrent logout requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := logout.Handler(mockUseCase, cookiesConfig)

		const numRequests = 10

		for i := range numRequests {
			token := "refresh_token_" + string(rune('0'+i))
			mockUseCase.EXPECT().
				LogoutUser(gomock.Any(), token).
				Return(nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				token := "refresh_token_" + string(rune('0'+idx))
				req := createRequest(t, token)
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
}
