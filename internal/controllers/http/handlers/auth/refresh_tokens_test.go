package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth"
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

func TestRefreshTokensHandler(t *testing.T) {
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
				Name:  auth.RefreshTokenCookieName,
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

	t.Run("successful token refresh", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)
		expectedTokens := createTestTokens()

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), validToken).
			Return(&expectedTokens, nil)

		req := createRequest(t, validToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())

		// Проверяем, что установлены правильные куки
		cookies := rr.Result().Cookies()
		assert.Len(t, cookies, 2)

		// Находим access token cookie
		var (
			accessTokenCookie  *http.Cookie
			refreshTokenCookie *http.Cookie
		)

		for _, cookie := range cookies {
			switch cookie.Name {
			case auth.AccessTokenCookieName:
				accessTokenCookie = cookie
			case auth.RefreshTokenCookieName:
				refreshTokenCookie = cookie
			}
		}

		require.NotNil(t, accessTokenCookie)
		require.NotNil(t, refreshTokenCookie)

		// Проверяем значения токенов
		assert.Equal(t, expectedTokens.AccessToken, accessTokenCookie.Value)
		assert.Equal(t, expectedTokens.RefreshToken, refreshTokenCookie.Value)

		// Проверяем настройки access token cookie
		assert.Equal(t, cookiesConfig.AccessToken.Path, accessTokenCookie.Path)
		assert.Equal(t, cookiesConfig.AccessToken.Domain, accessTokenCookie.Domain)
		assert.Equal(t, cookiesConfig.AccessToken.MaxAge, accessTokenCookie.MaxAge)
		assert.Equal(t, cookiesConfig.AccessToken.Secure, accessTokenCookie.Secure)
		assert.Equal(t, cookiesConfig.AccessToken.HTTPOnly, accessTokenCookie.HttpOnly)
		assert.Equal(t, int(cookiesConfig.AccessToken.SameSite), int(accessTokenCookie.SameSite))

		// Проверяем настройки refresh token cookie
		assert.Equal(t, cookiesConfig.RefreshToken.Path, refreshTokenCookie.Path)
		assert.Equal(t, cookiesConfig.RefreshToken.Domain, refreshTokenCookie.Domain)
		assert.Equal(t, cookiesConfig.RefreshToken.MaxAge, refreshTokenCookie.MaxAge)
		assert.Equal(t, cookiesConfig.RefreshToken.Secure, refreshTokenCookie.Secure)
		assert.Equal(t, cookiesConfig.RefreshToken.HTTPOnly, refreshTokenCookie.HttpOnly)
		assert.Equal(t, int(cookiesConfig.RefreshToken.SameSite), int(refreshTokenCookie.SameSite))
	})

	t.Run("unauthorized - missing refresh token cookie", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		// Запрос без куки
		req := createRequest(t, "")
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "http: named cookie not present")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("unauthorized - invalid JWT token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		invalidToken := "invalid_jwt_token"

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), invalidToken).
			Return(&domains.TokensDTO{}, customerrors.ErrInvalidJWT)

		req := createRequest(t, invalidToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("unauthorized - access token does not belong to refresh token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		refreshToken := "refresh_token_with_mismatched_access"

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), refreshToken).
			Return(&domains.TokensDTO{}, customerrors.ErrAccessTokenDoesNotBelongToRefreshToken)

		req := createRequest(t, refreshToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(
			t,
			rr.Body.String(),
			customerrors.ErrAccessTokenDoesNotBelongToRefreshToken.Error(),
		)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), validToken).
			Return(&domains.TokensDTO{}, errors.New("database connection failed"))

		req := createRequest(t, validToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("success with different cookie configurations", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			cookiesConfig config.CookiesConfig
		}{
			{
				name: "secure cookies",
				cookiesConfig: config.CookiesConfig{
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
						MaxAge:   2592000, // 30 дней
						Secure:   true,
						HTTPOnly: true,
						SameSite: http.SameSiteStrictMode,
					},
				},
			},
			{
				name: "development cookies (non-secure)",
				cookiesConfig: config.CookiesConfig{
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
			},
			{
				name: "session cookies (no MaxAge)",
				cookiesConfig: config.CookiesConfig{
					AccessToken: cookies.Config{
						Path:     "/",
						Domain:   "example.com",
						MaxAge:   0, // Session cookie
						Secure:   true,
						HTTPOnly: true,
						SameSite: http.SameSiteNoneMode,
					},
					RefreshToken: cookies.Config{
						Path:     "/",
						Domain:   "example.com",
						MaxAge:   0, // Session cookie
						Secure:   true,
						HTTPOnly: true,
						SameSite: http.SameSiteNoneMode,
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				handler := auth.RefreshTokensHandler(mockUseCase, tc.cookiesConfig)

				expectedTokens := createTestTokens()

				mockUseCase.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)

				req := createRequest(t, validToken)
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusNoContent, rr.Code)

				// Проверяем куки
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case auth.AccessTokenCookieName:
						accessTokenCookie = cookie
					case auth.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)

				// Проверяем настройки
				assert.Equal(t, tc.cookiesConfig.AccessToken.Path, accessTokenCookie.Path)
				assert.Equal(t, tc.cookiesConfig.AccessToken.Domain, accessTokenCookie.Domain)
				assert.Equal(t, tc.cookiesConfig.AccessToken.MaxAge, accessTokenCookie.MaxAge)
				assert.Equal(t, tc.cookiesConfig.AccessToken.Secure, accessTokenCookie.Secure)
				assert.Equal(t, tc.cookiesConfig.AccessToken.HTTPOnly, accessTokenCookie.HttpOnly)
				assert.Equal(
					t,
					int(tc.cookiesConfig.AccessToken.SameSite),
					int(accessTokenCookie.SameSite),
				)

				assert.Equal(t, tc.cookiesConfig.RefreshToken.Path, refreshTokenCookie.Path)
				assert.Equal(t, tc.cookiesConfig.RefreshToken.Domain, refreshTokenCookie.Domain)
				assert.Equal(t, tc.cookiesConfig.RefreshToken.MaxAge, refreshTokenCookie.MaxAge)
				assert.Equal(t, tc.cookiesConfig.RefreshToken.Secure, refreshTokenCookie.Secure)
				assert.Equal(t, tc.cookiesConfig.RefreshToken.HTTPOnly, refreshTokenCookie.HttpOnly)
				assert.Equal(
					t,
					int(tc.cookiesConfig.RefreshToken.SameSite),
					int(refreshTokenCookie.SameSite),
				)
			})
		}
	})

	t.Run("multiple refresh token cookies - first one is used", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		// Первый токен будет использован
		firstToken := "first_refresh_token"
		secondToken := "second_refresh_token"

		expectedTokens := createTestTokens()

		// Ожидаем, что будет использован первый токен
		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), firstToken).
			Return(&expectedTokens, nil)

		req := httptest.NewRequest(http.MethodPost, "/refresh", http.NoBody)

		// Добавляем две куки с одинаковым именем
		req.AddCookie(&http.Cookie{
			Name:  auth.RefreshTokenCookieName,
			Value: firstToken,
		})
		req.AddCookie(&http.Cookie{
			Name:  auth.RefreshTokenCookieName,
			Value: secondToken,
		})

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
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
			{"PATCH method", http.MethodPatch, http.StatusNoContent},
			{"DELETE method", http.MethodDelete, http.StatusNoContent},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// Arrange
				mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
				cookiesConfig := createCookiesConfig()
				handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

				expectedTokens := createTestTokens()

				mockUseCase.EXPECT().
					RefreshTokens(gomock.Any(), validToken).
					Return(&expectedTokens, nil)

				req := httptest.NewRequest(tc.method, "/refresh", http.NoBody)
				req.AddCookie(&http.Cookie{
					Name:  auth.RefreshTokenCookieName,
					Value: validToken,
				})

				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, tc.expectedStatus, rr.Code)
			})
		}
	})

	t.Run("very long refresh token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		// Создаем очень длинный токен (4096 символов)
		longToken := ""

		var longTokenSb495 strings.Builder
		for range 4096 {
			longTokenSb495.WriteString("a")
		}

		longToken += longTokenSb495.String()

		expectedTokens := createTestTokens()

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), longToken).
			Return(&expectedTokens, nil)

		req := createRequest(t, longToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("concurrent refresh requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		const numRequests = 10

		// Настраиваем мок на конкурентные вызовы
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

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				token := "token_" + strconv.Itoa(idx)
				req := createRequest(t, token)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				// Проверяем, что установлены правильные куки
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case auth.AccessTokenCookieName:
						accessTokenCookie = cookie
					case auth.RefreshTokenCookieName:
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

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("empty tokens from use case", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		expectedTokens := domains.TokensDTO{
			AccessToken:  "", // Пустые токены
			RefreshToken: "",
		}

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), validToken).
			Return(&expectedTokens, nil)

		req := createRequest(t, validToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен установить пустые куки
		assert.Equal(t, http.StatusNoContent, rr.Code)

		c := rr.Result().Cookies()
		assert.Len(t, c, 2)

		var (
			accessTokenCookie  *http.Cookie
			refreshTokenCookie *http.Cookie
		)

		for _, cookie := range c {
			switch cookie.Name {
			case auth.AccessTokenCookieName:
				accessTokenCookie = cookie
			case auth.RefreshTokenCookieName:
				refreshTokenCookie = cookie
			}
		}

		require.NotNil(t, accessTokenCookie)
		require.NotNil(t, refreshTokenCookie)
		assert.Equal(t, "", accessTokenCookie.Value)
		assert.Equal(t, "", refreshTokenCookie.Value)
	})

	t.Run("malformed JWT token", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := auth.RefreshTokensHandler(mockUseCase, cookiesConfig)

		malformedToken := "not.a.valid.jwt.token"

		mockUseCase.EXPECT().
			RefreshTokens(gomock.Any(), malformedToken).
			Return(&domains.TokensDTO{}, customerrors.ErrInvalidJWT)

		req := createRequest(t, malformedToken)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
	})
}
