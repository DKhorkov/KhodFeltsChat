package refresh_tokens_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/refresh_tokens"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/cookies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	validToken = "valid_token"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания конфига кук
	createCookiesConfig := func() config.CookiesConfig {
		return config.CookiesConfig{
			AccessToken: cookies.Config{
				Path:     "/",
				Domain:   "example.com",
				MaxAge:   900, // 15 минут
				Secure:   true,
				HTTPOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
			RefreshToken: cookies.Config{
				Path:     "/",
				Domain:   "example.com",
				MaxAge:   604800, // 7 дней
				Secure:   true,
				HTTPOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
		}
	}

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, refreshToken string) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/refresh", http.NoBody)

		if refreshToken != "" {
			cookie := &http.Cookie{
				Name:  login.RefreshTokenCookieName,
				Value: refreshToken,
			}
			req.AddCookie(cookie)
		}

		return req
	}

	// Вспомогательная функция для создания тестовых токенов
	createTestTokens := func() domains.TokensDTO {
		return domains.TokensDTO{
			AccessToken:  "new_access_token_123",
			RefreshToken: "new_refresh_token_456",
		}
	}

	cookiesConfig := createCookiesConfig()

	tests := []struct {
		name           string
		cookiesConfig  *config.CookiesConfig // nil означает использовать дефолтный
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful token refresh",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "new_access_token_123",
					RefreshToken: "new_refresh_token_456",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())

				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)

				assert.Equal(t, "new_access_token_123", accessTokenCookie.Value)
				assert.Equal(t, "new_refresh_token_456", refreshTokenCookie.Value)

				cc := createCookiesConfig()

				assert.Equal(t, cc.AccessToken.Path, accessTokenCookie.Path)
				assert.Equal(t, cc.AccessToken.Domain, accessTokenCookie.Domain)
				assert.Equal(t, cc.AccessToken.MaxAge, accessTokenCookie.MaxAge)
				assert.Equal(t, cc.AccessToken.Secure, accessTokenCookie.Secure)
				assert.Equal(t, cc.AccessToken.HTTPOnly, accessTokenCookie.HttpOnly)
				assert.Equal(t, int(cc.AccessToken.SameSite), int(accessTokenCookie.SameSite))

				assert.Equal(t, cc.RefreshToken.Path, refreshTokenCookie.Path)
				assert.Equal(t, cc.RefreshToken.Domain, refreshTokenCookie.Domain)
				assert.Equal(t, cc.RefreshToken.MaxAge, refreshTokenCookie.MaxAge)
				assert.Equal(t, cc.RefreshToken.Secure, refreshTokenCookie.Secure)
				assert.Equal(t, cc.RefreshToken.HTTPOnly, refreshTokenCookie.HttpOnly)
				assert.Equal(t, int(cc.RefreshToken.SameSite), int(refreshTokenCookie.SameSite))
			},
		},
		{
			name: "unauthorized - missing refresh token cookie",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "")
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "http: named cookie not present")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				c := rr.Result().Cookies()
				assert.Empty(t, c)
			},
		},
		{
			name: "unauthorized - invalid JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "invalid_jwt_token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					RefreshTokens(gomock.Any(), "invalid_jwt_token").
					Return(&domains.TokensDTO{}, customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				c := rr.Result().Cookies()
				assert.Empty(t, c)
			},
		},
		{
			name: "unauthorized - access token does not belong to refresh token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "refresh_token_with_mismatched_access")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					RefreshTokens(gomock.Any(), "refresh_token_with_mismatched_access").
					Return(&domains.TokensDTO{}, customerrors.ErrAccessTokenDoesNotBelongToRefreshToken)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(
					t,
					rr.Body.String(),
					customerrors.ErrAccessTokenDoesNotBelongToRefreshToken.Error(),
				)
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				c := rr.Result().Cookies()
				assert.Empty(t, c)
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
					RefreshTokens(gomock.Any(), validToken).
					Return(&domains.TokensDTO{}, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				c := rr.Result().Cookies()
				assert.Empty(t, c)
			},
		},
		{
			name: "secure cookies configuration",
			cookiesConfig: &config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   300,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   2592000,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "new_access_token_123",
					RefreshToken: "new_refresh_token_456",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)

				assert.Equal(t, "/api", accessTokenCookie.Path)
				assert.Equal(t, "secure.example.com", accessTokenCookie.Domain)
				assert.Equal(t, 300, accessTokenCookie.MaxAge)
				assert.True(t, accessTokenCookie.Secure)
				assert.True(t, accessTokenCookie.HttpOnly)
				assert.Equal(t, int(http.SameSiteStrictMode), int(accessTokenCookie.SameSite))

				assert.Equal(t, "/api", refreshTokenCookie.Path)
				assert.Equal(t, "secure.example.com", refreshTokenCookie.Domain)
				assert.Equal(t, 2592000, refreshTokenCookie.MaxAge)
				assert.True(t, refreshTokenCookie.Secure)
				assert.True(t, refreshTokenCookie.HttpOnly)
				assert.Equal(t, int(http.SameSiteStrictMode), int(refreshTokenCookie.SameSite))
			},
		},
		{
			name: "development cookies (non-secure) configuration",
			cookiesConfig: &config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   900,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   604800,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "new_access_token_123",
					RefreshToken: "new_refresh_token_456",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)

				assert.Equal(t, "/", accessTokenCookie.Path)
				assert.Equal(t, "localhost", accessTokenCookie.Domain)
				assert.Equal(t, 900, accessTokenCookie.MaxAge)
				assert.False(t, accessTokenCookie.Secure)
				assert.True(t, accessTokenCookie.HttpOnly)
				assert.Equal(t, int(http.SameSiteLaxMode), int(accessTokenCookie.SameSite))

				assert.Equal(t, "/", refreshTokenCookie.Path)
				assert.Equal(t, "localhost", refreshTokenCookie.Domain)
				assert.Equal(t, 604800, refreshTokenCookie.MaxAge)
				assert.False(t, refreshTokenCookie.Secure)
				assert.True(t, refreshTokenCookie.HttpOnly)
				assert.Equal(t, int(http.SameSiteLaxMode), int(refreshTokenCookie.SameSite))
			},
		},
		{
			name: "session cookies (no MaxAge) configuration",
			cookiesConfig: &config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "new_access_token_123",
					RefreshToken: "new_refresh_token_456",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)

				assert.Equal(t, 0, accessTokenCookie.MaxAge)
				assert.True(t, accessTokenCookie.Secure)
				assert.Equal(t, int(http.SameSiteNoneMode), int(accessTokenCookie.SameSite))

				assert.Equal(t, 0, refreshTokenCookie.MaxAge)
				assert.True(t, refreshTokenCookie.Secure)
				assert.Equal(t, int(http.SameSiteNoneMode), int(refreshTokenCookie.SameSite))
			},
		},
		{
			name: "multiple refresh token cookies - first one is used",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodPost, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{
					Name:  login.RefreshTokenCookieName,
					Value: "first_refresh_token",
				})
				req.AddCookie(&http.Cookie{
					Name:  login.RefreshTokenCookieName,
					Value: "second_refresh_token",
				})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "new_access_token_123",
					RefreshToken: "new_refresh_token_456",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), "first_refresh_token").
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "POST method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodPost, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{Name: login.RefreshTokenCookieName, Value: validToken})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "GET method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{Name: login.RefreshTokenCookieName, Value: validToken})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "PUT method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodPut, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{Name: login.RefreshTokenCookieName, Value: validToken})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "PATCH method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodPatch, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{Name: login.RefreshTokenCookieName, Value: validToken})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "DELETE method",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				req := httptest.NewRequest(http.MethodDelete, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{Name: login.RefreshTokenCookieName, Value: validToken})
				return req
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "very long refresh token",
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
				longToken := longTokenSb.String()
				expectedTokens := createTestTokens()
				m.EXPECT().
					RefreshTokens(gomock.Any(), longToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name: "empty tokens from use case",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, validToken)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{
					AccessToken:  "",
					RefreshToken: "",
				}
				m.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
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
			},
		},
		{
			name: "malformed JWT token",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createRequest(t, "not.a.valid.jwt.token")
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					RefreshTokens(gomock.Any(), "not.a.valid.jwt.token").
					Return(&domains.TokensDTO{}, customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			cc := cookiesConfig
			if tt.cookiesConfig != nil {
				cc = *tt.cookiesConfig
			}

			handler := refresh_tokens.Handler(mockUseCase, cc)

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

	t.Run("concurrent refresh requests", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := refresh_tokens.Handler(mockUseCase, cookiesConfig)

		const numRequests = 10

		for i := range numRequests {
			token := "token_" + strconv.Itoa(i)
			expectedTokens := domains.TokensDTO{
				AccessToken:  "access_" + strconv.Itoa(i),
				RefreshToken: "refresh_" + strconv.Itoa(i),
			}

			mockUseCase.EXPECT().
				RefreshTokens(gomock.Any(), token).
				Return(&expectedTokens, nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				token := "token_" + strconv.Itoa(idx)
				req := createRequest(t, token)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)
				assert.Equal(t, "access_"+strconv.Itoa(idx), accessTokenCookie.Value)
				assert.Equal(t, "refresh_"+strconv.Itoa(idx), refreshTokenCookie.Value)

				done <- true
			}(i)
		}

		for range numRequests {
			<-done
		}
	})
}
