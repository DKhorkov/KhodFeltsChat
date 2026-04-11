package login_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/auth/login"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/cookies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	createRequest := func(t *testing.T, requestBody io.Reader) *http.Request {
		t.Helper()

		return httptest.NewRequest(http.MethodPost, "/login", requestBody)
	}

	// Вспомогательная функция для создания LoginDTO
	createLoginDTO := func() domains.LoginDTO {
		return domains.LoginDTO{
			Email:    "test@example.com",
			Password: "SecurePassword123!",
		}
	}

	// Вспомогательная функция для создания тестовых токенов
	createTestTokens := func() domains.TokensDTO {
		return domains.TokensDTO{
			AccessToken:  "access_token_123",
			RefreshToken: "refresh_token_456",
		}
	}

	t.Run("successful login with email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedTokens := createTestTokens()

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&expectedTokens, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Empty(t, rr.Body.String())

		// Проверяем, что установлены правильные куки
		c := rr.Result().Cookies()
		assert.Len(t, c, 2)

		// Находим access token cookie
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

	t.Run("bad request - empty request body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// Пустое тело запроса
		req := createRequest(t, bytes.NewReader([]byte{}))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// Невалидный JSON
		invalidJSON := `{"email": "test@example.com", "password": invalid}`
		req := createRequest(t, bytes.NewReader([]byte(invalidJSON)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid character")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("bad request - missing required fields", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		testCases := []struct {
			name string
			json string
		}{
			{
				name: "missing email",
				json: `{"password": "password123"}`,
			},
			{
				name: "missing password",
				json: `{"email": "test@example.com"}`,
			},
			{
				name: "empty object",
				json: `{}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				// DTO будет создан с пустыми полями, валидация произойдет в use case
				var dto domains.LoginDTO

				err := json.Unmarshal([]byte(tc.json), &dto)
				require.NoError(t, err, "JSON should unmarshal even with missing fields")

				mockUseCase.EXPECT().
					LoginUser(gomock.Any(), dto).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)

				req := createRequest(t, bytes.NewReader([]byte(tc.json)))
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

				// Проверяем, что куки не установлены
				c := rr.Result().Cookies()
				assert.Empty(t, c)
			})
		}
	})

	t.Run("bad request - validation failed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := domains.LoginDTO{
			Email:    "invalid-email", // Невалидный email
			Password: "123",           // Слишком короткий пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrUserNotFound)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("forbidden - email not confirmed", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrEmailNotConfirmed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrEmailNotConfirmed.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

		// Проверяем, что куки не установлены
		c := rr.Result().Cookies()
		assert.Empty(t, c)
	})

	t.Run("unauthorized - wrong password", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrWrongPassword)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
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
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, errors.New("database connection failed"))

		req := createRequest(t, bytes.NewReader(requestBody))
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

	t.Run("login with different cookie configurations", func(t *testing.T) {
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
				handler := login.Handler(mockUseCase, tc.cookiesConfig)

				dto := createLoginDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				expectedTokens := createTestTokens()

				mockUseCase.EXPECT().
					LoginUser(gomock.Any(), dto).
					Return(&expectedTokens, nil)

				req := createRequest(t, bytes.NewReader(requestBody))
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
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
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

	t.Run("login with different email formats", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		testCases := []struct {
			name        string
			email       string
			expectError bool
		}{
			{
				name:        "simple email",
				email:       "user@example.com",
				expectError: false,
			},
			{
				name:        "email with plus",
				email:       "user+tag@example.com",
				expectError: false,
			},
			{
				name:        "email with dot",
				email:       "user.name@example.com",
				expectError: false,
			},
			{
				name:        "email with subdomain",
				email:       "user@sub.example.com",
				expectError: false,
			},
			{
				name:        "email with domain zone",
				email:       "user@example.co.uk",
				expectError: false,
			},
			{
				name:        "invalid email - missing @",
				email:       "userexample.com",
				expectError: true,
			},
			{
				name:        "invalid email - missing domain",
				email:       "user@",
				expectError: true,
			},
			{
				name:        "invalid email - spaces",
				email:       "user @example.com",
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				dto := domains.LoginDTO{
					Email:    tc.email,
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				if tc.expectError {
					mockUseCase.EXPECT().
						LoginUser(gomock.Any(), dto).
						Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusBadRequest, rr.Code)
				} else {
					expectedTokens := createTestTokens()

					mockUseCase.EXPECT().
						LoginUser(gomock.Any(), dto).
						Return(&expectedTokens, nil)

					req := createRequest(t, bytes.NewReader(requestBody))
					rr := httptest.NewRecorder()

					// Act
					handler.ServeHTTP(rr, req)

					// Assert
					assert.Equal(t, http.StatusNoContent, rr.Code)
				}
			})
		}
	})

	t.Run("concurrent login requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		const numRequests = 5

		// Каждый запрос с разными данными
		requestsData := make([]domains.LoginDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.LoginDTO{
				Email:    "user" + strconv.Itoa(i) + "@example.com",
				Password: "Password123!" + strconv.Itoa(i),
			}
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedTokens := domains.TokensDTO{
				AccessToken:  "access_" + strconv.Itoa(i),
				RefreshToken: "refresh_" + strconv.Itoa(i),
			}

			mockUseCase.EXPECT().
				LoginUser(gomock.Any(), requestsData[i]).
				Return(&expectedTokens, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, bytes.NewReader(requestBody))
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

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("login with extra fields in JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// JSON с дополнительными полями, которых нет в DTO
		jsonWithExtraFields := `{
			"email": "test@example.com",
			"password": "SecurePassword123!",
			"extraField": "should be ignored",
			"anotherExtra": 123,
			"rememberMe": true
		}`

		expectedTokens := createTestTokens()

		// Ожидаем, что только определенные поля будут в DTO
		expectedDTO := domains.LoginDTO{
			Email:    "test@example.com",
			Password: "SecurePassword123!",
		}

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), expectedDTO).
			Return(&expectedTokens, nil)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("login with null values", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// JSON с null значениями
		jsonWithNull := `{
			"email": null,
			"password": "SecurePassword123!"
		}`

		// При десериализации null в string станет пустой строкой
		var dto domains.LoginDTO

		err := json.Unmarshal([]byte(jsonWithNull), &dto)
		require.NoError(t, err)
		assert.Equal(t, "", dto.Email)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должна быть ошибка валидации
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("login with empty strings", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := domains.LoginDTO{
			Email:    "", // Пустой email
			Password: "", // Пустой пароль
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
	})

	t.Run("empty tokens from use case", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		dto := createLoginDTO()
		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Use case возвращает пустые токены (пограничный случай)
		expectedTokens := domains.TokensDTO{
			AccessToken:  "", // Пустые токены
			RefreshToken: "",
		}

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&expectedTokens, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
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
	})

	t.Run("rate limiting scenarios", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// Многократные попытки входа с неверным паролем
		dto := domains.LoginDTO{
			Email:    "test@example.com",
			Password: "WrongPassword123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Несколько попыток с неверным паролем
		const attempts = 3
		for range attempts {
			mockUseCase.EXPECT().
				LoginUser(gomock.Any(), dto).
				Return(&domains.TokensDTO{}, customerrors.ErrWrongPassword)
		}

		// Act & Assert - несколько попыток входа
		for range attempts {
			req := createRequest(t, bytes.NewReader(requestBody))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())

			// Куки не должны устанавливаться
			c := rr.Result().Cookies()
			assert.Empty(t, c)
		}
	})

	t.Run("login with case-sensitive email", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		testCases := []struct {
			name  string
			email string
		}{
			{
				name:  "lowercase email",
				email: "user@example.com",
			},
			{
				name:  "uppercase email",
				email: "USER@EXAMPLE.COM",
			},
			{
				name:  "mixed case email",
				email: "UsEr@ExAmPlE.CoM",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				dto := domains.LoginDTO{
					Email:    tc.email,
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				expectedTokens := createTestTokens()

				// В зависимости от реализации, email может быть case-insensitive
				mockUseCase.EXPECT().
					LoginUser(gomock.Any(), dto).
					Return(&expectedTokens, nil)

				req := createRequest(t, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()

				// Act
				handler.ServeHTTP(rr, req)

				// Assert - если use case нормализует email, то успешно
				assert.Equal(t, http.StatusNoContent, rr.Code)
			})
		}
	})

	t.Run("login with email containing special characters", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// Email с специальными символами до @ (обычно допустимы)
		dto := domains.LoginDTO{
			Email:    "user.name+tag@example.com",
			Password: "SecurePassword123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		expectedTokens := createTestTokens()

		mockUseCase.EXPECT().
			LoginUser(gomock.Any(), dto).
			Return(&expectedTokens, nil)

		req := createRequest(t, bytes.NewReader(requestBody))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})
}
